package buildinfo

import "runtime/debug"

// buildInfoWith constructs the toolchain metadata shape that fillFromBuildInfo
// reads, so the fallback logic can be tested without rebuilding the binary.
func buildInfoWith(settings map[string]string, mainVersion string) *debug.BuildInfo {
	build := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/jwogrady/echo", Version: mainVersion},
	}

	for key, value := range settings {
		build.Settings = append(build.Settings, debug.BuildSetting{Key: key, Value: value})
	}

	return build
}
