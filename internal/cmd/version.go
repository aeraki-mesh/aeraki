package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tetratelabs/run/pkg/version"
)

// cmdVersion prints the application version information using the Tetratelabs
// version package.
func cmdVersion(appName string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			version.Show(appName)
		},
	}
}
