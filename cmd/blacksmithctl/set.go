package main

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/cafebazaar/blacksmith/swagger/client/operations"
	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use: "set",
}

func NewSetVariablesNodesMacKey() *cobra.Command {
	mandatoryArgs := []string{
		"<mac>",
		"<key>",
		"<value>",
	}
	return &cobra.Command{
		Use: "node-key " + strings.Join(mandatoryArgs, " "),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args, mandatoryArgs)
		},
		Run: func(cmd *cobra.Command, args []string) {
			var mac, key, val = args[0], args[1], args[2]

			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			resp, err := c.Operations.PostVariablesNodesMacKey(&operations.PostVariablesNodesMacKeyParams{
				Context: ctx,
				Mac:     mac,
				Key:     key,
				Value:   val,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(formatOutput(resp.Payload))
		},
	}
}

func NewSetVariablesClusterKey() *cobra.Command {
	mandatoryArgs := []string{
		"<key>",
		"<value>",
	}
	return &cobra.Command{
		Use: "cluster-key " + strings.Join(mandatoryArgs, " "),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args, mandatoryArgs)
		},
		Run: func(cmd *cobra.Command, args []string) {
			var key, val = args[0], args[1]

			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			resp, err := c.Operations.PostVariablesClusterKey(&operations.PostVariablesClusterKeyParams{
				Context: ctx,
				Key:     key,
				Value:   val,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(formatOutput(resp.Payload))
		},
	}
}

func init() {
	RootCmd.AddCommand(setCmd)
	setCmd.AddCommand(NewSetVariablesNodesMacKey())
	setCmd.AddCommand(NewSetVariablesClusterKey())
	// setCmd.AddCommand(NewSetWorkspace())
}
