package version

import "runtime/debug"

var Value = "dev"

func String() string {
	if Value != "" && Value != "dev" {
		return Value
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}
