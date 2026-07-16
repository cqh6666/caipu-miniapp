package buildinfo

import "testing"

func TestCurrentReleaseIDTrimsAndFallsBack(t *testing.T) {
	original := ReleaseID
	t.Cleanup(func() { ReleaseID = original })

	ReleaseID = "  release-123  "
	if got := CurrentReleaseID(); got != "release-123" {
		t.Fatalf("CurrentReleaseID()=%q", got)
	}
	ReleaseID = "   "
	if got := CurrentReleaseID(); got != "dev" {
		t.Fatalf("empty CurrentReleaseID()=%q", got)
	}
}

func TestCurrentBuildIdentityNormalizesInjectedValues(t *testing.T) {
	originalReleaseID, originalGitCommit := ReleaseID, GitCommit
	originalBuildTime, originalGoToolchain := BuildTime, GoToolchain
	t.Cleanup(func() {
		ReleaseID, GitCommit = originalReleaseID, originalGitCommit
		BuildTime, GoToolchain = originalBuildTime, originalGoToolchain
	})

	ReleaseID = " release-123 "
	GitCommit = " abcdef123456 "
	BuildTime = " 2026-07-16T04:00:00Z "
	GoToolchain = " go1.26.5 "
	info := Current()
	if info.ReleaseID != "release-123" || info.GitCommit != "abcdef123456" || info.BuildTime != "2026-07-16T04:00:00Z" || info.GoToolchain != "go1.26.5" {
		t.Fatalf("Current() = %#v", info)
	}
}
