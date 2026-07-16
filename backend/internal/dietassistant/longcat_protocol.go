package dietassistant

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	longCatToolOpenTag  = "<longcat_tool_call>"
	longCatToolCloseTag = "</longcat_tool_call>"
	longCatArgKeyOpen   = "<longcat_arg_key>"
	longCatArgKeyClose  = "</longcat_arg_key>"
	longCatArgValOpen   = "<longcat_arg_value>"
	longCatArgValClose  = "</longcat_arg_value>"
)

func sanitizeStoredMessages(items []StoredMessage) []StoredMessage {
	if len(items) == 0 {
		return nil
	}
	result := make([]StoredMessage, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Role), "assistant") {
			item.Content = sanitizeAssistantVisibleContent(item.Content)
			if item.Content == "" {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

func sanitizeAssistantVisibleContent(content string) string {
	return strings.TrimSpace(stripLongCatToolCallBlocks(content))
}

func stripLongCatToolCallBlocks(content string) string {
	content = strings.TrimSpace(content)
	if content == "" || !strings.Contains(content, longCatToolOpenTag) {
		return content
	}

	var builder strings.Builder
	remaining := content
	for {
		start := strings.Index(remaining, longCatToolOpenTag)
		if start < 0 {
			builder.WriteString(remaining)
			break
		}
		builder.WriteString(remaining[:start])
		remaining = remaining[start+len(longCatToolOpenTag):]
		end := strings.Index(remaining, longCatToolCloseTag)
		if end < 0 {
			break
		}
		remaining = remaining[end+len(longCatToolCloseTag):]
	}
	return strings.TrimSpace(builder.String())
}

func parseLongCatToolCalls(content string) ([]openAIToolCall, bool) {
	content = strings.TrimSpace(content)
	if content == "" || !strings.Contains(content, longCatToolOpenTag) {
		return nil, false
	}

	remaining := content
	calls := make([]openAIToolCall, 0, 1)
	for index := 0; ; index += 1 {
		start := strings.Index(remaining, longCatToolOpenTag)
		if start < 0 {
			break
		}
		remaining = remaining[start+len(longCatToolOpenTag):]
		end := strings.Index(remaining, longCatToolCloseTag)
		if end < 0 {
			break
		}
		block := strings.TrimSpace(remaining[:end])
		remaining = remaining[end+len(longCatToolCloseTag):]

		call, ok := parseLongCatToolCallBlock(block, index)
		if ok {
			calls = append(calls, call)
		}
	}
	return normalizeToolCallIDs(calls), true
}

type longCatStreamFilter struct {
	pending     string
	block       strings.Builder
	inToolBlock bool
	markupFound bool
	visible     strings.Builder
	toolCalls   []openAIToolCall
	emitVisible func(string) error
}

func newLongCatStreamFilter(emitVisible func(string) error) *longCatStreamFilter {
	return &longCatStreamFilter{
		toolCalls:   make([]openAIToolCall, 0, 1),
		emitVisible: emitVisible,
	}
}

func (f *longCatStreamFilter) Push(delta string) error {
	if delta == "" {
		return nil
	}
	f.pending += delta
	for {
		if f.inToolBlock {
			end := strings.Index(f.pending, longCatToolCloseTag)
			if end < 0 {
				keep := tagPartialSuffixLen(f.pending, longCatToolCloseTag)
				if keep >= len(f.pending) {
					return nil
				}
				if err := f.writeToolBlock(f.pending[:len(f.pending)-keep]); err != nil {
					return err
				}
				f.pending = f.pending[len(f.pending)-keep:]
				return nil
			}

			if err := f.writeToolBlock(f.pending[:end]); err != nil {
				return err
			}
			call, ok := parseLongCatToolCallBlock(f.block.String(), len(f.toolCalls))
			if ok {
				f.toolCalls = append(f.toolCalls, call)
			}
			f.pending = f.pending[end+len(longCatToolCloseTag):]
			f.block.Reset()
			f.inToolBlock = false
			continue
		}

		start := strings.Index(f.pending, longCatToolOpenTag)
		if start >= 0 {
			if err := f.writeVisible(f.pending[:start]); err != nil {
				return err
			}
			f.markupFound = true
			f.pending = f.pending[start+len(longCatToolOpenTag):]
			f.inToolBlock = true
			f.block.Reset()
			continue
		}

		keep := tagPartialSuffixLen(f.pending, longCatToolOpenTag)
		if keep >= len(f.pending) {
			return nil
		}
		visible := f.pending[:len(f.pending)-keep]
		f.pending = f.pending[len(f.pending)-keep:]
		return f.writeVisible(visible)
	}
}

func (f *longCatStreamFilter) Flush() error {
	if f.inToolBlock {
		f.pending = ""
		f.block.Reset()
		f.inToolBlock = false
		return nil
	}
	if f.pending == "" {
		return nil
	}
	visible := f.pending
	f.pending = ""
	return f.writeVisible(visible)
}

func (f *longCatStreamFilter) writeVisible(delta string) error {
	if delta == "" {
		return nil
	}
	if f.visible.Len()+len(delta) > maxDietAssistantVisibleBytes {
		return dietAssistantLimitError("diet assistant visible content exceeded size limit", errVisibleTextTooLarge)
	}
	f.visible.WriteString(delta)
	if f.emitVisible == nil {
		return nil
	}
	return f.emitVisible(delta)
}

func (f *longCatStreamFilter) writeToolBlock(delta string) error {
	if delta == "" {
		return nil
	}
	if f.block.Len()+len(delta) > maxDietAssistantToolBlockBytes {
		return dietAssistantLimitError("diet assistant tool payload exceeded size limit", errToolPayloadTooLarge)
	}
	f.block.WriteString(delta)
	return nil
}

func (f *longCatStreamFilter) VisibleContent() string {
	if f == nil {
		return ""
	}
	return f.visible.String()
}

func (f *longCatStreamFilter) ToolCalls() []openAIToolCall {
	if f == nil || len(f.toolCalls) == 0 {
		return nil
	}
	return normalizeToolCallIDs(f.toolCalls)
}

func (f *longCatStreamFilter) MarkupFound() bool {
	return f != nil && f.markupFound
}

func tagPartialSuffixLen(value, tag string) int {
	if tag == "" {
		return 0
	}
	maxLen := len(tag) - 1
	if len(value) < maxLen {
		maxLen = len(value)
	}
	for length := maxLen; length > 0; length -= 1 {
		if strings.HasSuffix(value, tag[:length]) {
			return length
		}
	}
	return 0
}

func renameToolCallIDs(calls []openAIToolCall, prefix string) []openAIToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := append([]openAIToolCall{}, calls...)
	for index := range result {
		result[index].ID = fmt.Sprintf("%s_%d", prefix, index+1)
		if strings.TrimSpace(result[index].Type) == "" {
			result[index].Type = "function"
		}
	}
	return result
}

func parseLongCatToolCallBlock(block string, index int) (openAIToolCall, bool) {
	block = strings.TrimSpace(block)
	if block == "" {
		return openAIToolCall{}, false
	}

	toolName := block
	argsBody := ""
	if argIndex := strings.Index(block, longCatArgKeyOpen); argIndex >= 0 {
		toolName = strings.TrimSpace(block[:argIndex])
		argsBody = block[argIndex:]
	}
	if toolName == "" {
		return openAIToolCall{}, false
	}

	args := make(map[string]any)
	for {
		keyStart := strings.Index(argsBody, longCatArgKeyOpen)
		if keyStart < 0 {
			break
		}
		keyBody := argsBody[keyStart+len(longCatArgKeyOpen):]
		keyEnd := strings.Index(keyBody, longCatArgKeyClose)
		if keyEnd < 0 {
			break
		}
		key := strings.TrimSpace(keyBody[:keyEnd])
		valueBody := keyBody[keyEnd+len(longCatArgKeyClose):]
		valStart := strings.Index(valueBody, longCatArgValOpen)
		if valStart < 0 {
			break
		}
		valueBody = valueBody[valStart+len(longCatArgValOpen):]
		valEnd := strings.Index(valueBody, longCatArgValClose)
		if valEnd < 0 {
			break
		}
		value := strings.TrimSpace(valueBody[:valEnd])
		argsBody = valueBody[valEnd+len(longCatArgValClose):]
		if key != "" {
			args[key] = value
		}
	}

	return openAIToolCall{
		ID:   fmt.Sprintf("call_diet_assistant_longcat_%d", index+1),
		Type: "function",
		Function: openAIToolCallFunction{
			Name:      toolName,
			Arguments: normalizeLongCatToolArguments(toolName, args),
		},
	}, true
}

func normalizeLongCatToolArguments(toolName string, args map[string]any) map[string]any {
	normalized := make(map[string]any, len(args)+4)
	for key, value := range args {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		normalized[trimmed] = value
	}

	switch strings.TrimSpace(toolName) {
	case "get_recipe_count":
		ensureToolArg(normalized, "mealType", "all")
		ensureToolArg(normalized, "status", "all")
	case "search_recipes_by_name":
		if !hasNonEmptyToolArg(normalized, "keyword") {
			if value, ok := normalized["query"]; ok {
				normalized["keyword"] = value
			}
		}
		if !hasNonEmptyToolArg(normalized, "searchScope") {
			switch {
			case hasNonEmptyToolArg(normalized, "ingredientKeyword") && !hasNonEmptyToolArg(normalized, "titleKeyword") && !hasNonEmptyToolArg(normalized, "keyword"):
				normalized["searchScope"] = "ingredient"
			case hasNonEmptyToolArg(normalized, "titleKeyword") && !hasNonEmptyToolArg(normalized, "ingredientKeyword") && !hasNonEmptyToolArg(normalized, "keyword"):
				normalized["searchScope"] = "title"
			default:
				normalized["searchScope"] = "title_or_ingredient"
			}
		}
		ensureToolArg(normalized, "mealType", "all")
		ensureToolArg(normalized, "status", "all")
	case "get_recipe_by_id":
		if !hasNonEmptyToolArg(normalized, "recipeId") {
			if value, ok := normalized["id"]; ok {
				normalized["recipeId"] = value
			}
		}
	case "parse_and_add_recipe_from_url":
		if !hasNonEmptyToolArg(normalized, "url") {
			if value, ok := normalized["link"]; ok {
				normalized["url"] = value
			}
		}
		ensureToolArg(normalized, "mealType", "main")
		ensureToolArg(normalized, "status", "wishlist")
	}
	return normalized
}

func ensureToolArg(args map[string]any, key, fallback string) {
	if hasNonEmptyToolArg(args, key) {
		return
	}
	args[key] = fallback
}

func hasNonEmptyToolArg(args map[string]any, key string) bool {
	value, ok := args[key]
	if !ok || value == nil {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(value)) != ""
}

func lastUserMessageContent(messages []ChatMessage) string {
	for index := len(messages) - 1; index >= 0; index -= 1 {
		message := messages[index]
		if strings.EqualFold(strings.TrimSpace(message.Role), "user") {
			return strings.TrimSpace(message.Content)
		}
	}
	return ""
}

func buildURLOnlyParseToolCall(content string) (openAIToolCall, bool) {
	rawURL, ok := singleURLOnly(content)
	if !ok {
		return openAIToolCall{}, false
	}
	return openAIToolCall{
		ID:   "call_diet_assistant_url_parse",
		Type: "function",
		Function: openAIToolCallFunction{
			Name: "parse_and_add_recipe_from_url",
			Arguments: map[string]any{
				"url":      rawURL,
				"mealType": "main",
				"status":   "wishlist",
			},
		},
	}, true
}

func singleURLOnly(content string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) != 1 {
		return "", false
	}
	rawURL := strings.Trim(fields[0], "<>")
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if parsed.Host == "" {
		return "", false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return rawURL, true
	default:
		return "", false
	}
}
