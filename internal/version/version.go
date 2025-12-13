package version

import (
	"fmt"
	"time"
)

var (
	BuildDate   string // YYYY-MM-DD (UTC)
	BuildCommit string
	BuildBranch string
	BuildCI     string
)

var buildEpoch = time.Date(
	1999, time.January, 31,
	0, 0, 0, 0,
	time.UTC,
)

// VersionInfo describes the build metadata in structured form.
type VersionInfo struct {
	BuildID    int
	BuildDate  string
	Commit     string
	Branch     string
	CI         string
	Calculated bool
	Error      string
}

func CalculateBuildID() (int, error) {
	if BuildDate == "" {
		return 0, fmt.Errorf("BuildDate is empty")
	}

	t, err := time.ParseInLocation("2006-01-02", BuildDate, time.UTC)
	if err != nil {
		return 0, fmt.Errorf("invalid BuildDate %q: %w", BuildDate, err)
	}

	if t.Before(buildEpoch) {
		return 0, fmt.Errorf("BuildDate %s is before epoch", BuildDate)
	}

	// Using hours avoids DST issues; epoch and build date are both UTC.
	days := int(t.Sub(buildEpoch).Hours() / 24)
	return days, nil
}

// Info returns structured version information.
// Safe to call at any time.
func Info() VersionInfo {
	id, err := CalculateBuildID()

	info := VersionInfo{
		BuildDate: BuildDate,
		Commit:    BuildCommit,
		Branch:    BuildBranch,
		CI:        BuildCI,
	}

	if err != nil {
		info.Error = err.Error()
		return info
	}

	info.BuildID = id
	info.Calculated = true
	return info
}

// String returns a human-readable build string.
func String() string {
	info := Info()

	if !info.Calculated {
		return fmt.Sprintf("Build unknown (%s)", info.Error)
	}

	return fmt.Sprintf(
		"Build %d (%s) commit[%s] branch[%s] ci[%s]",
		info.BuildID,
		info.BuildDate,
		coalesce(info.Commit, "unknown"),
		coalesce(info.Branch, "unknown"),
		coalesce(info.CI, "local"),
	)
}

func coalesce(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
