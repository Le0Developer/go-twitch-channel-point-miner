package cmd

import (
	"fmt"
	"os"

	miner "github.com/le0developer/go-twitch-channel-point-miner/src"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "tcpm",
		Short: "Twitch Channel Point Miner",
		Long:  "An application for mining Twitch channel points.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./tcpm.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigType("yaml")
		viper.SetConfigName("tcpm")
		viper.AddConfigPath(".")
	}

	viper.SetDefault("users", []map[string]string{})
	viper.SetDefault("mine.points", true)
	viper.SetDefault("mine.raids", true)
	viper.SetDefault("mine.moments", true)
	viper.SetDefault("mine.watchtime", true)
	viper.SetDefault("mine.predictions", true)
	viper.SetDefault("predictions.min_points", 1_000)
	viper.SetDefault("predictions.max_bet", 50_000)
	viper.SetDefault("predictions.max_ratio", 2)
	viper.SetDefault("predictions.stealth", false)
	viper.SetDefault("predictions.strategy", miner.PredictionStrategyCautious)
	viper.SetDefault("predictions.min_data_points", 5)
	viper.SetDefault("points.concurrent_point_limit", 2)
	viper.SetDefault("points.concurrent_watch_limit", 0)
	viper.SetDefault("points.prioritize_streaks", true)
	viper.SetDefault("points.strategy", miner.MiningStrategyLeastPoints)
	viper.SetDefault("chat.only_live", true)
	viper.SetDefault("chat.follow_chat_spam", false)
	viper.SetDefault("persistent.file", "persistent.json")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
