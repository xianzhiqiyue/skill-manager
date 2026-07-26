package cmd

import (
	"strings"
	"testing"

	"github.com/skill-home/cli/internal/registry"
	"github.com/spf13/viper"
)

func TestRunFeedbackSubmitsStructuredFeedback(t *testing.T) {
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")
	t.Cleanup(viper.Reset)
	fake := &fakeRegistryClient{
		feedbackResp: &registry.SkillFeedback{FeedbackType: "suggestion", Status: "pending"},
	}
	restore := swapRegistryClientFactory(func() registryClient { return fake })
	defer restore()

	if err := runFeedback("@team/reviewer", &feedbackOptions{
		feedbackType: "suggestion",
		message:      "希望补充失败示例",
	}); err != nil {
		t.Fatalf("runFeedback returned error: %v", err)
	}
	if len(fake.feedbackCalls) != 1 {
		t.Fatalf("expected one feedback call, got %d", len(fake.feedbackCalls))
	}
	call := fake.feedbackCalls[0]
	if call.namespace != "team" || call.name != "reviewer" || call.req == nil {
		t.Fatalf("unexpected feedback call: %+v", call)
	}
	if call.req.FeedbackType != "suggestion" || call.req.Content != "希望补充失败示例" {
		t.Fatalf("unexpected feedback request: %+v", call.req)
	}
}

func TestRunFeedbackValidatesTypeAndLogin(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	err := runFeedback("@team/reviewer", &feedbackOptions{feedbackType: "issue", message: "有问题"})
	if err == nil || !strings.Contains(err.Error(), "未登录") {
		t.Fatalf("expected login error, got %v", err)
	}

	viper.Set("registry.api_key", "sk_test")
	err = runFeedback("@team/reviewer", &feedbackOptions{feedbackType: "rating", message: "有问题"})
	if err == nil || !strings.Contains(err.Error(), "反馈类型") {
		t.Fatalf("expected feedback type error, got %v", err)
	}
}

func TestRunRateReturnsMigrationMessage(t *testing.T) {
	err := runRate("@team/reviewer", &rateOptions{score: 5, comment: "good"})
	if err == nil || !strings.Contains(err.Error(), "feedback") {
		t.Fatalf("expected feedback migration message, got %v", err)
	}
}
