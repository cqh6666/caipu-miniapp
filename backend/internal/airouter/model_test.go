package airouter

import "testing"

func TestChatCompletionExtraForPersistenceNormalizesThinkingOptions(t *testing.T) {
	t.Parallel()

	got, err := ChatCompletionExtraForPersistence(map[string]any{
		"thinking_type":    "disabled",
		"reasoning_effort": "",
	}, EndpointModeChatCompletions)
	if err != nil {
		t.Fatalf("ChatCompletionExtraForPersistence() error = %v", err)
	}
	if got["thinking_type"] != "disabled" {
		t.Fatalf("thinking_type = %#v, want disabled", got["thinking_type"])
	}
	if _, ok := got["reasoning_effort"]; ok {
		t.Fatalf("reasoning_effort should be omitted when empty: %#v", got["reasoning_effort"])
	}
}

func TestChatCompletionExtraForPersistenceOmitsAutoThinking(t *testing.T) {
	t.Parallel()

	got, err := ChatCompletionExtraForPersistence(map[string]any{
		"thinking_type": "auto",
	}, EndpointModeChatCompletions)
	if err != nil {
		t.Fatalf("ChatCompletionExtraForPersistence() error = %v", err)
	}
	if _, ok := got["thinking_type"]; ok {
		t.Fatalf("thinking_type should be omitted for auto: %#v", got["thinking_type"])
	}
}

func TestChatCompletionExtraForPersistenceRejectsReasoningWithDisabledThinking(t *testing.T) {
	t.Parallel()

	_, err := ChatCompletionExtraForPersistence(map[string]any{
		"thinking_type":    "disabled",
		"reasoning_effort": "max",
	}, EndpointModeChatCompletions)
	if err == nil {
		t.Fatal("ChatCompletionExtraForPersistence() error = nil, want non-nil")
	}
}

func TestChatCompletionExtraForPersistenceDropsTextOptionsForImageEndpoint(t *testing.T) {
	t.Parallel()

	got, err := ChatCompletionExtraForPersistence(map[string]any{
		"thinking_type":    "enabled",
		"reasoning_effort": "max",
		"size":             "1024x1024",
	}, EndpointModeImagesGenerations)
	if err != nil {
		t.Fatalf("ChatCompletionExtraForPersistence() error = %v", err)
	}
	if _, ok := got["thinking_type"]; ok {
		t.Fatalf("thinking_type should be omitted for image endpoint: %#v", got["thinking_type"])
	}
	if _, ok := got["reasoning_effort"]; ok {
		t.Fatalf("reasoning_effort should be omitted for image endpoint: %#v", got["reasoning_effort"])
	}
	if got["size"] != "1024x1024" {
		t.Fatalf("size = %#v, want preserved image option", got["size"])
	}
}
