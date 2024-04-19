/*
Copyright © 2024 Metal toolbox authors <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/metal-toolbox/bioscfg/internal/model"
	"github.com/spf13/cobra"
)

var (
	args = &model.Args{}
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bioscfg",
	Short: "bioscfg remotely manages BIOS settings",
	Run: func(cmd *cobra.Command, _ []string) {
		if err := runWorker(cmd.Context(), args); err != nil {
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().
		StringVar(&args.ConfigFile, "config", "", "configuration file (default is $HOME/.bioscfg.yml)")

	rootCmd.PersistentFlags().
		StringVar(&args.LogLevel, "log-level", "info", "set logging level - debug, trace")

	rootCmd.PersistentFlags().
		BoolVarP(&args.EnableProfiling, "enable-pprof", "", false, "Enable profiling endpoint at: http://localhost:9091")

	rootCmd.PersistentFlags().
		StringVarP(&args.FacilityCode, "facility-code", "f", "", "The facility code this bioscfg instance is associated with")

	if err := rootCmd.MarkPersistentFlagRequired("facility-code"); err != nil {
		slog.Error("failed to mark required flag", "error", err)
		os.Exit(1)
	}
}
