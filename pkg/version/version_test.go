package version

import "testing"

func TestCalculateBuildID(t *testing.T) {
	tests := []struct {
		name      string
		date      string
		expected  int
		wantError bool
	}{
		{
			name:     "epoch date",
			date:     "2025-12-04",
			expected: 0,
		},
		{
			name:     "next day after epoch",
			date:     "2025-12-05",
			expected: 1,
		},
		{
			name:     "one year later",
			date:     "2026-12-04",
			expected: 365,
		},
		{
			name:     "date with leap years included",
			date:     "2032-12-04",
			expected: 2557,
		},
		{
			name:      "invalid format",
			date:      "invalid",
			wantError: true,
		},
		{
			name:      "empty date",
			date:      "",
			wantError: true,
		},
		{
			name:      "before epoch",
			date:      "2025-12-03",
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			old := BuildDate
			defer func() { BuildDate = old }()

			BuildDate = tt.date

			got, err := CalculateBuildID()

			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil (id=%d)", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.expected {
				t.Errorf("CalculateBuildID() = %d, want %d", got, tt.expected)
			}
		})
	}
}
