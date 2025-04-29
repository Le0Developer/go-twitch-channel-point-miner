package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	updateConfig bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&updateConfig, "update", "u", false, "Update the config file with new default values")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Init config file",
	Long:  "Initialize the config file. This will create a config file in the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		if updateConfig {
			cobra.CheckErr(viper.WriteConfig())
			cmd.Println("Config file updated successfully.")
			return
		}

		cobra.CheckErr(viper.WriteConfig())
		cmd.Println("Config file created successfully.")
	},
}
