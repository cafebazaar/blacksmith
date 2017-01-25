package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:           "blacksmithctl",
	Short:         "blacksmithctl is a client for blacksmith",
	Long:          "blacksmithctl is a client for blacksmith",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func httpRequest(method, p string, values url.Values) (*http.Response, error) {
	u, err := url.Parse("http://172.19.1.1:8000")
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, p)

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if values != nil {
		req.URL.RawQuery = values.Encode()
	}

	timeout, _ := RootCmd.Flags().GetInt("timeout")
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	req = req.WithContext(ctx)
	c := http.Client{}
	return c.Do(req)
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	RootCmd.PersistentFlags().IntP("timeout", "t", 2, "timeout in seconds")
	RootCmd.PersistentFlags().StringP("output", "o", "yaml", "output format. One of: yaml|json")

	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".blacksmith") // name of config file (without extension)
	viper.AddConfigPath("$HOME")       // adding home directory as first search path
	viper.AutomaticEnv()               // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func checkArgs(args, mandatoryArgs []string) error {
	if got, want := len(args), len(mandatoryArgs); got != want {
		if got < want {
			isOrAre := "are"
			if want-got == 1 {
				isOrAre = "is"
			}
			return fmt.Errorf("%s %s missing", strings.Join(mandatoryArgs[got:], " "), isOrAre)
		} else if got > want {
			return fmt.Errorf("%d extra args was given", got-want)
		}
	}
	return nil
}
