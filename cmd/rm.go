package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cafebazaar/blacksmith/swagger/client/operations"
	"github.com/spf13/cobra"
)

// rmCmd represents the set command
var rmCmd = &cobra.Command{
	Use: "rm",
}

func NewRmVariablesNodesMacKey() *cobra.Command {
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
			var mac, key = args[0], args[1]

			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			_, err := c.Operations.DeleteVariablesNodesMacKey(&operations.DeleteVariablesNodesMacKeyParams{
				Context: ctx,
				Mac:     mac,
				Key:     key,
			})
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
		},
	}
}

func NewRmVariablesClusterKey() *cobra.Command {
	mandatoryArgs := []string{
		"<key>",
	}
	return &cobra.Command{
		Use: "cluster-key " + strings.Join(mandatoryArgs, " "),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args, mandatoryArgs)
		},
		Run: func(cmd *cobra.Command, args []string) {
			var key = args[0]

			c := newSwaggerClient()
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			_, err := c.Operations.DeleteVariablesClusterKey(&operations.DeleteVariablesClusterKeyParams{
				Context: ctx,
				Key:     key,
			})
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
		},
	}
}

func init() {
	RootCmd.AddCommand(rmCmd)
	rmCmd.AddCommand(NewRmVariablesNodesMacKey())
	rmCmd.AddCommand(NewRmVariablesClusterKey())
}
