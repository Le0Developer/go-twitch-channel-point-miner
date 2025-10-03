package cmd

import (
	"os"

	miner "github.com/le0developer/go-twitch-channel-point-miner/src"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	autoLogin bool
)

var (
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the Twitch Channel Point Miner",
		Long:  "Run the Twitch Channel Point Miner with the specified configuration.",

		Run: func(cmd *cobra.Command, args []string) {
			usersObjs := viper.Get("users").([]any)
			if len(usersObjs) == 0 {
				cmd.PrintErrln("No users found in the configuration file.")
				if !autoLogin {
					cmd.PrintErrln("Please run the login command to add users.")
					os.Exit(1)
					return
				}
				save = true
				loginCmd.Execute()
			}

			options := miner.Options{
				MinePoints:            viper.GetBool("mine.points"),
				PrioritizeStreaks:     viper.GetBool("points.prioritize_streaks"),
				ConcurrentPointLimit:  viper.GetInt("points.concurrent_point_limit"),
				ConcurrentWatchLimit:  viper.GetInt("points.concurrent_watch_limit"),
				MiningStrategy:        miner.MiningStrategy(viper.GetString("points.strategy")),
				MineRaids:             viper.GetBool("mine.raids"),
				MineMoments:           viper.GetBool("mine.moments"),
				MinePredictions:       viper.GetBool("mine.predictions"),
				PredictionsMinPoints:  viper.GetInt("predictions.min_points"),
				PredictionsMaxBet:     viper.GetInt("predictions.max_bet"),
				PredictionsMaxRatio:   viper.GetInt("predictions.max_ratio"),
				PredictionsStealth:    viper.GetBool("predictions.stealth"),
				PredictionsStrategy:   miner.PredictionStrategy(viper.GetString("predictions.strategy")),
				PredictionsDataPoints: viper.GetInt("predictions.min_data_points"),
				MineWatchtime:         viper.GetBool("mine.watchtime"),
				WatchTimeOnlyLive:     viper.GetBool("chat.only_live"),
				FollowChatSpam:        viper.GetBool("chat.follow_chat_spam"),
				StreamerPriority:      map[string]int{},
				DebugWebhook:          viper.GetString("debug.webhook"),
				PersistentFile:        viper.GetString("persistent.file"),
				PrometheusEnabled:     viper.GetBool("prometheus.enabled"),
				PrometheusPort:        viper.GetInt("prometheus.port"),
			}
			addFollowers := viper.GetBool("streamers.follows")
			instance := miner.NewMiner(options)
			users := []*miner.User{}
			for _, user := range usersObjs {
				userObj := user.(map[string]any)
				user := miner.NewUser(userObj["name"].(string), userObj["token"].(string))
				users = append(users, user)
				instance.AddUser(user)
				if addFollowers {
					instance.AddStreamersFromFollows(user)
				}
			}
			priorities, ok := viper.Get("streamers.streamers").(map[string]any)
			if ok {
				streamers := make([]string, 0, len(priorities))
				for k := range priorities {
					streamers = append(streamers, k)
					options.StreamerPriority[k] = priorities[k].(int)
				}
				for _, user := range users {
					instance.BulkAddStreamers(user, streamers)
				}
			}
			err := instance.Run()
			cobra.CheckErr(err)
		},
	}
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVarP(&autoLogin, "login", "l", false, "Automatically login if no users are found")
}
