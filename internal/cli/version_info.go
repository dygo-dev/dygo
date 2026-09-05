package cli

import (
	"runtime/debug"
	"strings"
)

var version = "dev"
var readBuildInfo = debug.ReadBuildInfo

func currentVersion() string {
	if cliVersion := strings.TrimSpace(version); cliVersion != "" && cliVersion != "dev" {
		return cliVersion
	}
	buildInfo, ok := readBuildInfo()
	if ok {
		buildVersion := dygoBuildVersion(buildInfo)
		if buildVersion != "" && buildVersion != "(devel)" {
			return buildVersion
		}
	}
	return "dev"
}

const dygoModulePath = "github.com/hapyco/dygo"

func dygoBuildVersion(buildInfo *debug.BuildInfo) string {
	if buildInfo == nil {
		return ""
	}
	if buildInfo.Main.Path == dygoModulePath {
		return strings.TrimSpace(buildInfo.Main.Version)
	}
	for _, dependency := range buildInfo.Deps {
		if dependency == nil || dependency.Path != dygoModulePath {
			continue
		}
		if dependency.Replace != nil && strings.TrimSpace(dependency.Replace.Version) != "" {
			return strings.TrimSpace(dependency.Replace.Version)
		}
		return strings.TrimSpace(dependency.Version)
	}
	return ""
}

func dygoVersionForNew() string {
	return currentVersion()
}
