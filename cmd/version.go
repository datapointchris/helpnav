package cmd

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/datapointchris/goselfupdate"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// buildVersion reports the running binary's version. `go install pkg@latest`
// applies no ldflags but does stamp the module version into build info, so
// without this fallback every installed copy identifies as a dev build and
// `helpnav update` refuses to run.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	return resolveVersion(version, info.Main.Version)
}

func resolveVersion(ldflagsVersion, moduleVersion string) string {
	if ldflagsVersion != "dev" && ldflagsVersion != "" {
		return ldflagsVersion
	}
	// Go stamps a VCS-derived pseudo-version onto local `go build` output and that
	// string is valid semver, which is why the release check lives in
	// goselfupdate rather than being re-derived here.
	if !goselfupdate.IsReleaseVersion(moduleVersion) {
		return ldflagsVersion
	}
	return strings.TrimPrefix(moduleVersion, "v")
}

var versionCmd = &cobra.Command{
	Use:     "version",
	GroupID: groupTool,
	Short:   "Print helpnav version information",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()
		if _, err := fmt.Fprintf(out, "helpnav %s\n", buildVersion()); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "  commit: %s\n", commit); err != nil {
			return err
		}
		_, err := fmt.Fprintf(out, "  built:  %s\n", date)
		return err
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
