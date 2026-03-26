package mcp

import (
	"testing"
	"time"
)

func TestDetectFeedbackType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Rejection phrase",
			content:  "I reject this approach",
			expected: "rejection",
		},
		{
			name:     "Not correct phrase",
			content:  "This is not correct",
			expected: "rejection",
		},
		{
			name:     "Approval phrase",
			content:  "Looks good to me, approve",
			expected: "approval",
		},
		{
			name:     "LGTM phrase",
			content:  "LGTM",
			expected: "approval",
		},
		{
			name:     "Escalation phrase",
			content:  "This is urgent, escalate",
			expected: "escalation",
		},
		{
			name:     "General feedback",
			content:  "Please check the formatting",
			expected: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFeedbackType(tt.content)
			if result != tt.expected {
				t.Errorf("detectFeedbackType(%q) = %q; want %q", tt.content, result, tt.expected)
			}
		})
	}
}

func TestIsRejectionFeedback(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		fbType   string
		expected bool
	}{
		{
			name:     "Explicit rejection type",
			content:  "Some feedback",
			fbType:   "rejection",
			expected: true,
		},
		{
			name:     "Spanish rejection phrase",
			content:  "Esto no es correcto",
			fbType:   "general",
			expected: true,
		},
		{
			name:     "English rejection phrase",
			content:  "This is not correct",
			fbType:   "general",
			expected: true,
		},
		{
			name:     "Wait phrase",
			content:  "Espera, eso no está bien",
			fbType:   "general",
			expected: true,
		},
		{
			name:     "Hold on phrase",
			content:  "Hold on, this doesn't look right",
			fbType:   "general",
			expected: true,
		},
		{
			name:     "Positive feedback",
			content:  "This looks great!",
			fbType:   "general",
			expected: false,
		},
		{
			name:     "LGTM",
			content:  "LGTM, proceed",
			fbType:   "general",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRejectionFeedback(tt.content, tt.fbType)
			if result != tt.expected {
				t.Errorf("isRejectionFeedback(%q, %q) = %v; want %v", tt.content, tt.fbType, result, tt.expected)
			}
		})
	}
}

func TestFeedbackDetectionPatterns(t *testing.T) {
	rejectionPhrases := []string{
		"no es correcto",
		"not correct",
		"wrong",
		"incorrect",
		"esto no",
		"that's wrong",
		"not right",
		"doesn't look right",
		"no estoy de acuerdo",
		"i disagree",
		"reject",
		"rechazo",
		"no debería",
		"should not",
		"please stop",
		"hold on",
		"espera",
		"wait",
		"hold",
		"revisa esto",
		"review this",
	}

	positivePhrases := []string{
		"looks good",
		"lgtm",
		"approved",
		"perfecto",
		"bien hecho",
		"excelente",
		"correcto",
	}

	for _, phrase := range rejectionPhrases {
		if !isRejectionFeedback(phrase, "general") {
			t.Errorf("Expected %q to be detected as rejection", phrase)
		}
	}

	for _, phrase := range positivePhrases {
		if isRejectionFeedback(phrase, "general") {
			t.Errorf("Expected %q to NOT be detected as rejection", phrase)
		}
	}
}

func TestDodCheckResult_Structure(t *testing.T) {
	result := &DodCheckResult{
		WorkflowID: "test-workflow-1",
		Passed:     true,
		CheckedAt:  timeNow(),
		Checks: []DodCheckItem{
			{
				Check:   "workflow_completed",
				Passed:  true,
				Message: "Workflow completed",
			},
			{
				Check:   "all_steps_completed",
				Passed:  true,
				Message: "All 5 steps completed",
			},
		},
	}

	if result.WorkflowID != "test-workflow-1" {
		t.Errorf("expected WorkflowID=test-workflow-1, got %s", result.WorkflowID)
	}

	if !result.Passed {
		t.Error("expected Passed=true")
	}

	if len(result.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(result.Checks))
	}
}

func TestPlanRevision_Tracking(t *testing.T) {
	result := &DodCheckResult{
		WorkflowID: "test-workflow-1",
		Passed:     false,
		CheckedAt:  timeNow(),
		Checks: []DodCheckItem{
			{
				Check:   "workflow_completed",
				Passed:  false,
				Message: "Workflow not completed",
			},
		},
	}

	if result.Passed {
		t.Error("expected result to show failure")
	}
}

type DodCheckResult struct {
	WorkflowID string
	Checks     []DodCheckItem
	Passed     bool
	CheckedAt  time.Time
}

type DodCheckItem struct {
	Check   string
	Passed  bool
	Message string
}

func timeNow() time.Time {
	return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
}
