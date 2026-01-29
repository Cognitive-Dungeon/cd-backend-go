package version

import (
	"testing"
)

func TestString(t *testing.T) {
	origTag, origBuild, origCommit, origDirty := Tag, BuildNum, Commit, DirtyStr
	defer func() {
		Tag, BuildNum, Commit, DirtyStr = origTag, origBuild, origCommit, origDirty
	}()

	tests := []struct {
		name     string
		tag      string
		build    string
		commit   string
		dirty    string
		expected string
	}{
		{
			name:     "clean release build",
			tag:      "v1.0.0",
			build:    "3890",
			commit:   "a1b2c3d",
			dirty:    "false",
			expected: "v1.0.0 Build 3890 [a1b2c3d]",
		},
		{
			name:     "dirty dev build",
			tag:      "v0.1.0",
			build:    "56",
			commit:   "f9e8d7c",
			dirty:    "true",
			expected: "v0.1.0 Build 56 [f9e8d7c] (dirty)",
		},
		{
			name:     "unknown/default state",
			tag:      "v0.0.0",
			build:    "0",
			commit:   "unknown",
			dirty:    "false",
			expected: "v0.0.0 Build 0 [unknown]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Tag = tt.tag
			BuildNum = tt.build
			Commit = tt.commit
			DirtyStr = tt.dirty

			got := String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGet_DirtyFlagParsing(t *testing.T) {
	// Проверяем, что строковый "true"/"false" корректно превращается в bool
	origDirty := DirtyStr
	defer func() { DirtyStr = origDirty }()

	tests := []struct {
		inputStr string
		wantBool bool
	}{
		{"true", true},
		{"false", false},
		{"random", false}, // Любой мусор кроме "true" должен быть false
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.inputStr, func(t *testing.T) {
			DirtyStr = tt.inputStr
			info := Get()
			if info.IsDirty != tt.wantBool {
				t.Errorf("Get().IsDirty for %q = %v, want %v", tt.inputStr, info.IsDirty, tt.wantBool)
			}
		})
	}
}
