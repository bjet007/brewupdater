package brewupdater

import "runtime/debug"

// version is set by GoReleaser through -ldflags "-X".
var version = ""

const develVersion = "(devel)"

// resolveVersion prefers the ldflag, then the module version from the build info, then "dev".
// buildInfo is nil when the binary carries none.
func resolveVersion(ldflag string, buildInfo *debug.BuildInfo) string {
	if ldflag != "" {
		return ldflag
	}
	if buildInfo != nil && buildInfo.Main.Version != "" && buildInfo.Main.Version != develVersion {
		return buildInfo.Main.Version
	}
	return "dev"
}

func readBuildInfo() *debug.BuildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	return info
}
