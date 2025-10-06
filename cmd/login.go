package cmd

import (
	miner "github.com/le0developer/go-twitch-channel-point-miner/src"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	username string
	save     bool
)

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringVarP(&username, "username", "u", "", "Twitch username")
	must(loginCmd.MarkFlagRequired("username"))
	loginCmd.Flags().BoolVarP(&save, "save", "s", false, "Save the token to the config file")
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Twitch",
	Long:  "Login to Twitch. This will start a OAuth2 flow to get a token.",
	Run: func(cmd *cobra.Command, args []string) {
		session := miner.NewLoginSession()
		code, err := session.GetCode()
		cobra.CheckErr(err)

		cmd.Println("Please open the following URL in your browser:")
		cmd.Println("  https://www.twitch.tv/activate")
		cmd.Println("And enter the following code:")
		cmd.Println("  " + code)
		cmd.Println("Waiting for authentication...")
		token, err := session.WaitForToken()
		cobra.CheckErr(err)

		cmd.Println("Token:", token)
		cmd.Println("Add it to your config file:")
		cmd.Println("")
		cmd.Println("users:")
		cmd.Println("  - name: " + username)
		cmd.Println("    token: " + token)

		if save {
			users := viper.Get("users").([]any)
			users = append(users, map[string]string{
				"name":  username,
				"token": token,
			})

			viper.Set("users", users)
			cobra.CheckErr(viper.WriteConfig())
			cmd.Println("Config file updated.")
		}
	},
}
