package domain

import "testing"

func TestParseAction(t *testing.T) {
	tests := []struct {
		input    string
		expected ActionType
	}{
		{"MOVE", ActionMove},
		{"move", ActionMove},
		{"Move", ActionMove},
		{"ATTACK", ActionAttack},
		{"WAIT", ActionWait},
		{"UNKNOWN_ACTION", ActionUnknown},
		{"", ActionUnknown},
	}

	for _, tt := range tests {
		result := ParseAction(tt.input)
		if result != tt.expected {
			t.Errorf("ParseAction(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestActionType_String(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected string
	}{
		{ActionMove, "MOVE"},
		{ActionAttack, "ATTACK"},
		{ActionUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.action.String(); got != tt.expected {
			t.Errorf("ActionType(%d).String() = %q, want %q", tt.action, got, tt.expected)
		}
	}
}
