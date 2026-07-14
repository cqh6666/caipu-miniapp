package dietassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func dietAssistantTools() []openAITool {
	return []openAITool{
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "get_recipe_count",
				Description: "查询当前美食库中符合条件的菜谱数量。",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"mealType": map[string]any{
							"type":        "string",
							"description": "餐别过滤：breakfast=早餐，main=正餐，all=全部。",
							"enum":        []string{"breakfast", "main", "all"},
						},
						"status": map[string]any{
							"type":        "string",
							"description": "状态过滤：wishlist=想吃，done=吃过，all=全部。",
							"enum":        []string{"wishlist", "done", "all"},
						},
					},
					"required": []string{"mealType", "status"},
				},
			},
		},
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "parse_and_add_recipe_from_url",
				Description: "根据 B 站或小红书菜谱链接解析内容，提取食材和步骤，并保存为当前空间的一道菜谱。用户只发送链接或明确要求保存链接菜谱时调用。",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"url": map[string]any{
							"type":        "string",
							"description": "B 站或小红书菜谱链接，必须是用户提供的原始 URL。",
						},
						"mealType": map[string]any{
							"type":        "string",
							"description": "餐别：breakfast=早餐，main=正餐。无法判断时用 main。",
							"enum":        []string{"breakfast", "main"},
						},
						"status": map[string]any{
							"type":        "string",
							"description": "状态：wishlist=想吃，done=吃过。无法判断时用 wishlist。",
							"enum":        []string{"wishlist", "done"},
						},
					},
					"required": []string{"url", "mealType", "status"},
				},
			},
		},
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "get_recipe_by_id",
				Description: "根据菜谱 ID 获取当前空间内的一道菜谱详情，包括菜名、餐别、状态、食材、做法步骤、备注和来源链接。用户提供菜谱 ID 并要求查看详情时调用。",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"recipeId": map[string]any{
							"type":        "string",
							"description": "菜谱 ID，必须来自用户提供或前文工具结果中的 id 字段。",
						},
					},
					"required": []string{"recipeId"},
				},
			},
		},
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "search_recipes_by_name",
				Description: "按菜谱名或食材模糊查询当前空间的菜谱。用户询问是否已有某道菜、查找菜谱、按名称搜索或查找某种食材能做什么时调用。",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"keyword": map[string]any{
							"type":        "string",
							"description": "菜名或食材关键词，例如“番茄”“鸡胸肉”。优先使用该字段。",
						},
						"searchScope": map[string]any{
							"type":        "string",
							"description": "搜索范围：title=只按菜名，ingredient=只按食材，title_or_ingredient=菜名或食材。无法判断时用 title_or_ingredient。",
							"enum":        []string{"title", "ingredient", "title_or_ingredient"},
						},
						"titleKeyword": map[string]any{
							"type":        "string",
							"description": "兼容字段：只按菜名搜索的关键词。优先使用 keyword + searchScope。",
						},
						"ingredientKeyword": map[string]any{
							"type":        "string",
							"description": "兼容字段：只按食材搜索的关键词。优先使用 keyword + searchScope。",
						},
						"mealType": map[string]any{
							"type":        "string",
							"description": "餐别过滤：breakfast=早餐，main=正餐，all=全部。",
							"enum":        []string{"breakfast", "main", "all"},
						},
						"status": map[string]any{
							"type":        "string",
							"description": "状态过滤：wishlist=想吃，done=吃过，all=全部。",
							"enum":        []string{"wishlist", "done", "all"},
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "最多返回数量，默认 5，最大 10。",
						},
					},
					"required": []string{"keyword", "searchScope", "mealType", "status"},
				},
			},
		},
	}
}

func (s *Service) executeTool(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	name := strings.TrimSpace(call.Function.Name)
	switch name {
	case "get_recipe_count":
		return s.executeGetRecipeCount(ctx, chatCtx, call)
	case "search_recipes_by_name":
		return s.executeSearchRecipesByName(ctx, chatCtx, call)
	case "get_recipe_by_id":
		return s.executeGetRecipeByID(ctx, chatCtx, call)
	case "parse_and_add_recipe_from_url":
		return s.executeParseAndAddRecipeFromURL(ctx, chatCtx, call)
	default:
		return map[string]any{
			"ok":    false,
			"error": "unknown tool: " + name,
		}
	}
}

func buildToolMutation(name string, result map[string]any) *StreamMutation {
	switch name {
	case "parse_and_add_recipe_from_url":
		recipe, ok := result["recipe"].(RecipeToolItem)
		if !ok || strings.TrimSpace(recipe.ID) == "" {
			return nil
		}
		return &StreamMutation{
			Type:        "recipe_created",
			RecipeID:    strings.TrimSpace(recipe.ID),
			RecipeTitle: strings.TrimSpace(recipe.Title),
			MealType:    strings.TrimSpace(recipe.MealType),
			Status:      strings.TrimSpace(recipe.Status),
		}
	default:
		return nil
	}
}

func toolResultFailed(result map[string]any) bool {
	value, ok := result["ok"]
	if !ok {
		return false
	}
	if passed, ok := value.(bool); ok {
		return !passed
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprint(value)), "false")
}

func toolStatusMessage(name, stage string) string {
	displayName := toolDisplayName(name)
	switch stage {
	case "start":
		switch name {
		case "get_recipe_count":
			return "正在统计美食库"
		case "search_recipes_by_name":
			return "正在查找菜谱"
		case "get_recipe_by_id":
			return "正在读取菜谱详情"
		case "parse_and_add_recipe_from_url":
			return "正在解析链接并保存食材"
		default:
			return "正在调用" + displayName
		}
	case "done":
		switch name {
		case "get_recipe_count":
			return "已完成菜谱统计"
		case "search_recipes_by_name":
			return "已完成菜谱查找"
		case "get_recipe_by_id":
			return "已读取菜谱详情"
		case "parse_and_add_recipe_from_url":
			return "已解析并保存食材"
		default:
			return displayName + "调用完成"
		}
	case "error":
		switch name {
		case "get_recipe_count":
			return "菜谱统计失败，正在整理说明"
		case "search_recipes_by_name":
			return "菜谱查找失败，正在整理说明"
		case "get_recipe_by_id":
			return "菜谱详情读取失败，正在整理说明"
		case "parse_and_add_recipe_from_url":
			return "链接解析保存失败，正在整理说明"
		default:
			return displayName + "调用失败，正在整理说明"
		}
	default:
		return displayName
	}
}

func toolDisplayName(name string) string {
	switch name {
	case "get_recipe_count":
		return "美食库统计"
	case "search_recipes_by_name":
		return "菜谱查找"
	case "get_recipe_by_id":
		return "菜谱详情"
	case "parse_and_add_recipe_from_url":
		return "链接解析保存"
	default:
		if strings.TrimSpace(name) == "" {
			return "工具"
		}
		return strings.TrimSpace(name)
	}
}

func (s *Service) executeGetRecipeCount(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	if s.countRecipes == nil {
		return map[string]any{"ok": false, "error": "recipe count tool is not configured"}
	}
	if chatCtx.UserID <= 0 {
		return map[string]any{"ok": false, "error": "user is required"}
	}
	if chatCtx.KitchenID <= 0 {
		return map[string]any{"ok": false, "error": "current kitchen is required"}
	}

	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	mealType := normalizeToolEnum(fmt.Sprint(args["mealType"]), "all")
	status := normalizeToolEnum(fmt.Sprint(args["status"]), "all")
	if !isAllowedRecipeCountMealType(mealType) {
		return map[string]any{"ok": false, "error": "invalid mealType: " + mealType}
	}
	if !isAllowedRecipeCountStatus(status) {
		return map[string]any{"ok": false, "error": "invalid status: " + status}
	}

	input := RecipeCountInput{
		UserID:    chatCtx.UserID,
		KitchenID: chatCtx.KitchenID,
		MealType:  emptyIfAll(mealType),
		Status:    emptyIfAll(status),
	}
	count, err := s.countRecipes(ctx, input)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{
		"ok":        true,
		"count":     count,
		"mealType":  mealType,
		"status":    status,
		"kitchenId": chatCtx.KitchenID,
	}
}

func (s *Service) executeGetRecipeByID(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	if s.getRecipeByID == nil {
		return map[string]any{"ok": false, "error": "recipe detail tool is not configured"}
	}
	if chatCtx.UserID <= 0 {
		return map[string]any{"ok": false, "error": "user is required"}
	}
	if chatCtx.KitchenID <= 0 {
		return map[string]any{"ok": false, "error": "current kitchen is required"}
	}

	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	recipeID := truncateRunes(toolStringArg(args, "recipeId"), 120)
	if recipeID == "" {
		recipeID = truncateRunes(toolStringArg(args, "id"), 120)
	}
	if recipeID == "" {
		return map[string]any{"ok": false, "error": "recipeId is required"}
	}

	item, err := s.getRecipeByID(ctx, RecipeGetInput{
		UserID:    chatCtx.UserID,
		KitchenID: chatCtx.KitchenID,
		RecipeID:  recipeID,
	})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	return map[string]any{
		"ok":        true,
		"kitchenId": chatCtx.KitchenID,
		"recipeId":  recipeID,
		"recipe":    item,
	}
}

func (s *Service) executeParseAndAddRecipeFromURL(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	if s.createFromURL == nil {
		return map[string]any{"ok": false, "error": "recipe url create tool is not configured"}
	}
	if chatCtx.UserID <= 0 {
		return map[string]any{"ok": false, "error": "user is required"}
	}
	if chatCtx.KitchenID <= 0 {
		return map[string]any{"ok": false, "error": "current kitchen is required"}
	}

	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	rawURL := truncateRunes(toolStringArg(args, "url"), 500)
	if rawURL == "" {
		return map[string]any{"ok": false, "error": "url is required"}
	}
	mealType := normalizeToolEnum(fmt.Sprint(args["mealType"]), "main")
	status := normalizeToolEnum(fmt.Sprint(args["status"]), "wishlist")
	if mealType != "breakfast" && mealType != "main" {
		mealType = "main"
	}
	if status != "wishlist" && status != "done" {
		status = "wishlist"
	}

	result, err := s.createFromURL(ctx, RecipeFromURLInput{
		UserID:    chatCtx.UserID,
		KitchenID: chatCtx.KitchenID,
		URL:       rawURL,
		MealType:  mealType,
		Status:    status,
	})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	return map[string]any{
		"ok":                   true,
		"message":              "链接已解析，菜谱和食材已保存到美食库。",
		"kitchenId":            chatCtx.KitchenID,
		"recipe":               result.Recipe,
		"source":               result.Source,
		"sourceDetail":         result.SourceDetail,
		"summaryMode":          result.SummaryMode,
		"mainIngredients":      result.MainIngredients,
		"secondaryIngredients": result.SecondaryIngredients,
		"stepsCount":           result.StepsCount,
		"warnings":             result.Warnings,
	}
}

func (s *Service) executeSearchRecipesByName(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	if s.searchRecipes == nil {
		return map[string]any{"ok": false, "error": "recipe search tool is not configured"}
	}
	if chatCtx.UserID <= 0 {
		return map[string]any{"ok": false, "error": "user is required"}
	}
	if chatCtx.KitchenID <= 0 {
		return map[string]any{"ok": false, "error": "current kitchen is required"}
	}

	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	keyword := truncateRunes(toolStringArg(args, "keyword"), 80)
	searchScope := normalizeToolEnum(fmt.Sprint(args["searchScope"]), "title_or_ingredient")
	titleKeyword := truncateRunes(toolStringArg(args, "titleKeyword"), 80)
	ingredientKeyword := truncateRunes(toolStringArg(args, "ingredientKeyword"), 80)
	if keyword == "" {
		switch {
		case titleKeyword != "":
			keyword = titleKeyword
			searchScope = "title"
		case ingredientKeyword != "":
			keyword = ingredientKeyword
			searchScope = "ingredient"
		default:
			return map[string]any{"ok": false, "error": "keyword is required"}
		}
	}
	if !isAllowedRecipeSearchScope(searchScope) {
		return map[string]any{"ok": false, "error": "invalid searchScope: " + searchScope}
	}
	mealType := normalizeToolEnum(fmt.Sprint(args["mealType"]), "all")
	status := normalizeToolEnum(fmt.Sprint(args["status"]), "all")
	if !isAllowedRecipeCountMealType(mealType) {
		return map[string]any{"ok": false, "error": "invalid mealType: " + mealType}
	}
	if !isAllowedRecipeCountStatus(status) {
		return map[string]any{"ok": false, "error": "invalid status: " + status}
	}

	limit := normalizeToolLimit(args["limit"], 5, 10)
	items, err := s.searchRecipes(ctx, RecipeSearchInput{
		UserID:            chatCtx.UserID,
		KitchenID:         chatCtx.KitchenID,
		Keyword:           keyword,
		SearchScope:       searchScope,
		TitleKeyword:      titleKeyword,
		IngredientKeyword: ingredientKeyword,
		MealType:          emptyIfAll(mealType),
		Status:            emptyIfAll(status),
		Limit:             limit,
	})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	return map[string]any{
		"ok":                true,
		"keyword":           keyword,
		"searchScope":       searchScope,
		"titleKeyword":      titleKeyword,
		"ingredientKeyword": ingredientKeyword,
		"mealType":          mealType,
		"status":            status,
		"limit":             limit,
		"count":             len(items),
		"items":             items,
	}
}

func parseToolArguments(value any) (map[string]any, error) {
	switch v := value.(type) {
	case string:
		return parseToolArgumentBytes([]byte(v))
	case map[string]any:
		return v, nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return parseToolArgumentBytes(data)
	}
}

func parseToolArgumentBytes(data []byte) (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal(data, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}
	return args, nil
}

func normalizeToolCallIDs(calls []openAIToolCall) []openAIToolCall {
	result := append([]openAIToolCall{}, calls...)
	for index := range result {
		if strings.TrimSpace(result[index].ID) == "" {
			result[index].ID = fmt.Sprintf("call_diet_assistant_%d", index+1)
		}
		if strings.TrimSpace(result[index].Type) == "" {
			result[index].Type = "function"
		}
	}
	return result
}

func normalizeToolEnum(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "<nil>" {
		return fallback
	}
	return value
}

func normalizeToolLimit(value any, fallback, max int) int {
	limit := 0
	switch v := value.(type) {
	case float64:
		limit = int(v)
	case int:
		limit = v
	case int64:
		limit = int(v)
	case json.Number:
		parsed, err := v.Int64()
		if err == nil {
			limit = int(parsed)
		}
	default:
		text := strings.TrimSpace(fmt.Sprint(v))
		if text != "" && text != "<nil>" {
			_, _ = fmt.Sscanf(text, "%d", &limit)
		}
	}
	if limit <= 0 {
		limit = fallback
	}
	if max > 0 && limit > max {
		limit = max
	}
	return limit
}

func isAllowedRecipeCountMealType(value string) bool {
	switch value {
	case "all", "breakfast", "main":
		return true
	default:
		return false
	}
}

func isAllowedRecipeCountStatus(value string) bool {
	switch value {
	case "all", "wishlist", "done":
		return true
	default:
		return false
	}
}

func isAllowedRecipeSearchScope(value string) bool {
	switch value {
	case "title", "ingredient", "title_or_ingredient":
		return true
	default:
		return false
	}
}

func emptyIfAll(value string) string {
	if value == "all" {
		return ""
	}
	return value
}

func toolStringArg(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "<nil>" {
		return ""
	}
	return text
}

func truncateRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
