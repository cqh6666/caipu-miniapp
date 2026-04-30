package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.longcat.chat/openai/v1"
const defaultModel = "LongCat-2.0-Preview"

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Tools       []toolSpec    `json:"tools,omitempty"`
	ToolChoice  any           `json:"tool_choice,omitempty"`
	Stream      bool          `json:"stream"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type toolSpec struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int             `json:"index"`
		FinishReason string          `json:"finish_reason"`
		Message      responseMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type responseMessage struct {
	Role      string     `json:"role"`
	Content   any        `json:"content"`
	ToolCalls []toolCall `json:"tool_calls"`
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

func main() {
	baseURL := flag.String("base-url", getenv("LONGCAT_BASE_URL", defaultBaseURL), "OpenAI-compatible base URL")
	model := flag.String("model", getenv("LONGCAT_MODEL", defaultModel), "model name")
	question := flag.String("question", "请调用 get_recipe_count 工具查询正餐菜谱数量，然后根据工具结果回答。", "user question")
	maxTokens := flag.Int("max-tokens", 1000, "max_tokens")
	temperature := flag.Float64("temperature", 0.2, "temperature")
	forceTool := flag.Bool("force-tool", false, "force the model to call get_recipe_count")
	printRaw := flag.Bool("raw", true, "print pretty JSON responses")
	flag.Parse()

	apiKey := strings.TrimSpace(getenv("LONGCAT_API_KEY", os.Getenv("OPENAI_API_KEY")))
	if apiKey == "" {
		exitf("missing LONGCAT_API_KEY; example: LONGCAT_API_KEY='***' go run ./cmd/longcat-tool-probe")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	tools := []toolSpec{recipeCountTool()}
	messages := []chatMessage{
		{Role: "system", Content: "You are a helpful recipe assistant. When a tool is relevant, call it before answering."},
		{Role: "user", Content: strings.TrimSpace(*question)},
	}

	toolChoice := any("auto")
	if *forceTool {
		toolChoice = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": "get_recipe_count",
			},
		}
	}

	firstReq := chatRequest{
		Model:       strings.TrimSpace(*model),
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  toolChoice,
		Stream:      false,
		MaxTokens:   *maxTokens,
		Temperature: *temperature,
	}
	firstResp, firstRaw, err := postChat(client, *baseURL, apiKey, firstReq)
	if err != nil {
		exitf("first chat request failed: %v", err)
	}
	fmt.Println("=== first response ===")
	if *printRaw {
		fmt.Println(prettyJSON(firstRaw))
	}

	if len(firstResp.Choices) == 0 {
		exitf("no choices returned")
	}
	assistant := firstResp.Choices[0].Message
	if len(assistant.ToolCalls) == 0 {
		fmt.Println("result: no tool_calls returned. The API may ignore tools, or the model chose to answer directly.")
		return
	}
	for index := range assistant.ToolCalls {
		if strings.TrimSpace(assistant.ToolCalls[index].ID) == "" {
			assistant.ToolCalls[index].ID = fmt.Sprintf("call_probe_%d", index+1)
		}
	}

	fmt.Printf("result: tool_calls returned (%d). Tool calling is likely supported.\n", len(assistant.ToolCalls))

	messages = append(messages, chatMessage{
		Role:      valueOrDefault(assistant.Role, "assistant"),
		Content:   assistant.Content,
		ToolCalls: assistant.ToolCalls,
	})
	for _, call := range assistant.ToolCalls {
		content, err := executeTool(call)
		if err != nil {
			content = map[string]any{
				"ok":    false,
				"error": err.Error(),
			}
		}
		messages = append(messages, chatMessage{
			Role:       "tool",
			Content:    mustJSON(content),
			ToolCallID: call.ID,
			Name:       call.Function.Name,
		})
	}

	finalReq := chatRequest{
		Model:       strings.TrimSpace(*model),
		Messages:    messages,
		Stream:      false,
		MaxTokens:   *maxTokens,
		Temperature: *temperature,
	}
	finalResp, finalRaw, err := postChat(client, *baseURL, apiKey, finalReq)
	if err != nil {
		exitf("final chat request failed: %v", err)
	}

	fmt.Println("=== final response ===")
	if *printRaw {
		fmt.Println(prettyJSON(finalRaw))
	}
	if len(finalResp.Choices) > 0 {
		fmt.Printf("final answer: %s\n", stringContent(finalResp.Choices[0].Message.Content))
	}
}

func recipeCountTool() toolSpec {
	return toolSpec{
		Type: "function",
		Function: toolFunction{
			Name:        "get_recipe_count",
			Description: "查询当前美食库里某个餐别的菜谱数量。这个探针返回固定模拟数据。",
			Parameters: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"mealType": map[string]any{
						"type":        "string",
						"description": "餐别：breakfast=早餐，main=正餐，all=全部。",
						"enum":        []string{"breakfast", "main", "all"},
					},
				},
				"required": []string{"mealType"},
			},
		},
	}
}

func executeTool(call toolCall) (map[string]any, error) {
	if call.Function.Name != "get_recipe_count" {
		return nil, fmt.Errorf("unknown tool: %s", call.Function.Name)
	}
	args, err := parseArguments(call.Function.Arguments)
	if err != nil {
		return nil, err
	}
	mealType := strings.TrimSpace(fmt.Sprint(args["mealType"]))
	counts := map[string]int{
		"breakfast": 8,
		"main":      23,
		"all":       31,
	}
	count, ok := counts[mealType]
	if !ok {
		return nil, fmt.Errorf("unsupported mealType: %s", mealType)
	}
	return map[string]any{
		"ok":       true,
		"mealType": mealType,
		"count":    count,
		"source":   "local-probe-fixture",
	}, nil
}

func postChat(client *http.Client, baseURL, apiKey string, payload chatRequest) (chatResponse, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return chatResponse{}, nil, err
	}
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return chatResponse{}, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return chatResponse{}, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return chatResponse{}, data, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return chatResponse{}, data, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result chatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return chatResponse{}, data, err
	}
	if result.Error != nil && strings.TrimSpace(result.Error.Message) != "" {
		return result, data, errors.New(result.Error.Message)
	}
	return result, data, nil
}

func parseArguments(value any) (map[string]any, error) {
	switch v := value.(type) {
	case string:
		return parseArgumentBytes([]byte(v))
	case map[string]any:
		return v, nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return parseArgumentBytes(data)
	}
}

func parseArgumentBytes(data []byte) (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal(data, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments %q: %w", string(data), err)
	}
	return args, nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func stringContent(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf(`{"ok":false,"error":%q}`, err.Error())
	}
	return string(data)
}

func prettyJSON(data []byte) string {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return strings.TrimSpace(string(data))
	}
	pretty, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return strings.TrimSpace(string(data))
	}
	return string(pretty)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
