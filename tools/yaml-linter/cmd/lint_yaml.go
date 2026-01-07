package cmd

import (
	"github.com/spf13/cobra"
)

const lintYAMLDesc = "Run the yaml linter"

type lintYAMLOptions struct {
	rootOptions *rootOptions
	testPrefix  string
}

func newlintYAMLCmd(rootOptions *rootOptions) *cobra.Command {
	o := &lintYAMLOptions{
		rootOptions: rootOptions,
	}
	cmd := &cobra.Command{
		Use:   "read-tests SERVICES_DIR",
		Short: lintYAMLDesc,
		Long:  lintYAMLDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return o.run(args)
		},
	}
	return cmd
}

func (o *lintYAMLOptions) run(args []string) error {
	return nil
}
