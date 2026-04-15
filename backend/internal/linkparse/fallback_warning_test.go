package linkparse

import "testing"

func TestBuildAISummaryFallbackWarningUsesUpstreamMessage(t *testing.T) {
	t.Parallel()

	got := buildAISummaryFallbackWarning(assertErr(`{"error":{"message":"未提供令牌 (request id: abc123)","type":"new_api_error"}}`))
	want := "AI 总结失败：未提供令牌；已回退到规则整理。"
	if got != want {
		t.Fatalf("buildAISummaryFallbackWarning() = %q, want %q", got, want)
	}
}

type staticError string

func (e staticError) Error() string {
	return string(e)
}

func assertErr(message string) error {
	return staticError(message)
}
