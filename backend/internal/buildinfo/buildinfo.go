package buildinfo

import (
	"runtime"
	"strings"
)

// ReleaseID is replaced by the deployment build through -ldflags -X.
var (
	ReleaseID   = "dev"
	GitCommit   = "unknown"
	BuildTime   = "unknown"
	GoToolchain = ""
)

type Info struct {
	ReleaseID   string `json:"releaseId"`
	GitCommit   string `json:"gitCommit"`
	BuildTime   string `json:"buildTime"`
	GoToolchain string `json:"goToolchain"`
}

func Current() Info {
	return Info{
		ReleaseID:   normalized(ReleaseID, "dev"),
		GitCommit:   normalized(GitCommit, "unknown"),
		BuildTime:   normalized(BuildTime, "unknown"),
		GoToolchain: normalized(GoToolchain, runtime.Version()),
	}
}

func CurrentReleaseID() string {
	return Current().ReleaseID
}

func normalized(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
