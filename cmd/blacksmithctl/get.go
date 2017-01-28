package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cafebazaar/blacksmith/swagger/client/operations"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use: "get",
}

func NewGetVariablesNodesMacKey() *cobra.Command {
	mandatoryArgs := []string{
		"<mac>",
		"<key>",
	}
	return &cobra.Command{
		Use: "node-key " + strings.Join(mandatoryArgs, " "),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args, mandatoryArgs)
		},
		Run: func(cmd *cobra.Command, args []string) {
			mac, key := args[0], args[1]

			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			resp, err := c.Operations.GetVariablesNodesMacKey(&operations.GetVariablesNodesMacKeyParams{
				Context: ctx,
				Mac:     mac,
				Key:     key,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(formatOutput(resp.Payload))
		},
	}
}

func NewGetVariablesNodesMac() *cobra.Command {
	mandatoryArgs := []string{
		"<mac>",
	}
	return &cobra.Command{
		Use: "node-keys " + strings.Join(mandatoryArgs, " "),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args, mandatoryArgs)
		},
		Run: func(cmd *cobra.Command, args []string) {
			mac := args[0]

			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			resp, err := c.Operations.GetVariablesNodesMac(&operations.GetVariablesNodesMacParams{
				Context: ctx,
				Mac:     mac,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(formatOutput(resp.Payload))
		},
	}
}

func NewGetVariablesClusterKey() *cobra.Command {
	mandatoryArgs := []string{
		"<key>",
	}
	return &cobra.Command{
		Use: "cluster-key " + strings.Join(mandatoryArgs, " "),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args, mandatoryArgs)
		},
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]

			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			resp, err := c.Operations.GetVariablesClusterKey(&operations.GetVariablesClusterKeyParams{
				Context: ctx,
				Key:     key,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(formatOutput(resp.Payload))
		},
	}
}

func NewGetVariablesCluster() *cobra.Command {
	mandatoryArgs := []string{}
	return &cobra.Command{
		Use: "cluster-keys " + strings.Join(mandatoryArgs, " "),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args, mandatoryArgs)
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			resp, err := c.Operations.GetVariablesCluster(&operations.GetVariablesClusterParams{
				Context: ctx,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(formatOutput(resp.Payload))
		},
	}
}

func NewGetNodesCmd() *cobra.Command {
	return &cobra.Command{
		Use: "nodes",
		Run: func(cmd *cobra.Command, args []string) {
			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			resp, err := c.Operations.GetNodes(&operations.GetNodesParams{Context: ctx})
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(formatOutput(resp.Payload))
		},
	}
}

func NewGetWorkspaceCmd() *cobra.Command {
	return &cobra.Command{
		Use: "workspace",
		Run: func(cmd *cobra.Command, args []string) {
			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			resp, err := c.Operations.GetWorkspace(&operations.GetWorkspaceParams{Context: ctx})
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(formatOutput(resp.Payload))
		},
	}
}

func init() {
	RootCmd.AddCommand(getCmd)
	getCmd.AddCommand(NewGetNodesCmd())
	getCmd.AddCommand(NewGetWorkspaceCmd())

	getCmd.AddCommand(NewGetVariablesNodesMac())
	getCmd.AddCommand(NewGetVariablesNodesMacKey())

	getCmd.AddCommand(NewGetVariablesClusterKey())
	getCmd.AddCommand(NewGetVariablesCluster())
}
