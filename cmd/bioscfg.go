package cmd

import (
	"fmt"
	"os"

	"github.com/metal-toolbox/bioscfg/internal/controllers"
	"github.com/metal-toolbox/bioscfg/internal/controllers/kind"
	"github.com/spf13/cobra"
)

// bioscfgCmd represents the bioscfg command
var bioscfgCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the BiosCfg Controller",
	Run: func(cmd *cobra.Command, _ []string) {
		err := controllers.Run(cmd.Context(), kind.BiosCfg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(bioscfgCmd)
}
