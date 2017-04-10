package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cafebazaar/blacksmith/swagger/client/operations"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use: "install",
}

func NewInstallNodeCmd() *cobra.Command {
	mandatoryArgs := []string{
		"<mac>",
	}
	return &cobra.Command{
		Use: "node " + strings.Join(mandatoryArgs, " "),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args, mandatoryArgs)
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := newSwaggerClient()
			mac := args[0]
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			_, err := c.Operations.PostWorkspaceInstallMac(&operations.PostWorkspaceInstallMacParams{
				Mac:     mac,
				Context: ctx,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
		},
	}
}

func init() {
	RootCmd.AddCommand(installCmd)
	installCmd.AddCommand(NewInstallNodeCmd())
	// installCmd.AddCommand(NewInstallWorkspaceCmd())
}
