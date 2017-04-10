package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cafebazaar/blacksmith/swagger/client/operations"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use: "update",
}

func NewUpdateWorkspaceCmd() *cobra.Command {
	return &cobra.Command{
		Use: "workspaces",
		Run: func(cmd *cobra.Command, args []string) {
			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			_, err := c.Operations.PostWorkspaces(&operations.PostWorkspacesParams{Context: ctx})
			if err != nil {
				fmt.Println(err)
				return
			}
		},
	}
}

func NewUpdateNodeCmd() *cobra.Command {
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
			_, err := c.Operations.PostWorkspaceUpdateMac(&operations.PostWorkspaceUpdateMacParams{
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
	RootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(NewUpdateNodeCmd())
	updateCmd.AddCommand(NewUpdateWorkspaceCmd())
}
