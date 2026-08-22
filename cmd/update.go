package cmd

import (
	"github.com/datapointchris/goselfupdate"
	"github.com/datapointchris/goselfupdate/cobracmd"
)

// updateConfig describes where helpnav's releases come from. Shared by the
// `update` command and the check in Execute, so the notice cannot advertise a
// release that update refuses to install.
func updateConfig() goselfupdate.Config {
	return goselfupdate.Config{
		Owner:   "datapointchris",
		Repo:    "helpnav",
		Binary:  "helpnav",
		Version: buildVersion(),
	}
}

func init() {
	updateCmd := cobracmd.New(updateConfig())
	updateCmd.GroupID = groupTool
	rootCmd.AddCommand(updateCmd)
}
