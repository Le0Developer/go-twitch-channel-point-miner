package miner

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Miner struct {
	Options       Options
	WebsocketPool *WebsocketPool

	DefaultUser *User
	Users       map[string]*User
	Streamers   map[string]*Streamer
	Predictions map[string]*Prediction
	Persistent  *PersistentState

	SpadeUrl string

	Lock sync.Mutex
}

func (miner *Miner) AddUser(user *User) error {
	user.Miner = miner

	miner.Users[user.Username] = user
	if miner.DefaultUser == nil {
		miner.DefaultUser = user
	}

	if user.ID == "" {
		user.ID, _ = user.GraphQL.GetSteamerID(user.Username)
	}

	follows, err := user.GraphQL.GetFollows()
	if err != nil {
		return err
	}

	// using a waitgroup here because AddStreamer does a network request per streamer
	streamerch := make(chan *Streamer, len(follows))
	for _, follow := range follows {
		go func(follow string) {
			streamer := miner.AddStreamer(follow, user)
			err := user.GraphQL.LoadChannelPoints(streamer)
			if err != nil {
				fmt.Println("Error loading channel points", err)
			}
			streamerch <- streamer
		}(follow)
	}

	streamers := map[string]*Streamer{}
	for streamer := range streamerch {
		streamers[streamer.Username] = streamer
		if _, ok := miner.Streamers[streamer.Username]; !ok {
			miner.Streamers[streamer.Username] = streamer
		}
		if len(streamers) == len(follows) {
			close(streamerch)
		}
	}
	user.Streamers = streamers

	return nil
}

func (miner *Miner) AddStreamer(username string, user *User) *Streamer {
	if existing, ok := miner.Streamers[username]; ok {
		return existing
	}

	id, err := user.GraphQL.GetSteamerID(username)
	if err != nil {
		panic(err)
	}

	streamer := &Streamer{
		Username:      username,
		ID:            id,
		Points:        map[*User]int{},
		GotPointsOnce: map[*User]bool{},
	}

	return streamer
}

func (miner *Miner) GetStreamerByID(streamerID string) *Streamer {
	for _, streamer := range miner.Streamers {
		if streamer.ID == streamerID {
			return streamer
		}
	}
	return nil
}

func (miner *Miner) GetUsersForStreamer(id string) []*User {
	users := []*User{}
	for _, user := range miner.Users {
		for _, streamer := range user.Streamers {
			if streamer.ID == id {
				users = append(users, user)
			}
		}
	}
	return users
}

func (miner *Miner) SubscribeToTopics() {
	for _, user := range miner.Users {
		channelPointsTopic := WebsocketTopic{Topic: "community-points-user-v1", User: user}
		miner.WebsocketPool.ListenTopic(&channelPointsTopic)

		if miner.Options.MinePredictions {
			predictionTopic := WebsocketTopic{Topic: "predictions-user-v1", User: user}
			miner.WebsocketPool.ListenTopic(&predictionTopic)
		}
	}
	for _, streamer := range miner.Streamers {
		if miner.Options.RequiresStreamActivity() {
			videoPlaybackTopic := WebsocketTopic{Topic: "video-playback-by-id", Streamer: streamer}
			miner.WebsocketPool.ListenTopic(&videoPlaybackTopic)
		}

		if miner.Options.MineRaids {
			raidTopic := WebsocketTopic{Topic: "raid", Streamer: streamer}
			streamer.LiveTopics = append(streamer.LiveTopics, &raidTopic)
		}
		if miner.Options.MineMoments {
			momentTopic := WebsocketTopic{Topic: "community-moments-channel-v1", Streamer: streamer}
			streamer.LiveTopics = append(streamer.LiveTopics, &momentTopic)
		}
		if miner.Options.MinePredictions {
			predictionTopic := WebsocketTopic{Topic: "predictions-channel-v1", Streamer: streamer}
			streamer.LiveTopics = append(streamer.LiveTopics, &predictionTopic)
		}
	}
}

func (miner *Miner) Run() error {
	fmt.Println("Starting miner")

	err := miner.UpdateVersions()
	if err != nil {
		return err
	}

	if miner.Options.MineWatchtime {
		for _, user := range miner.Users {
			fmt.Println("Connecting to chat for user", user.Username)
			user.ConnectToChat()
		}
	}

	onMessage := miner.OnMessage
	miner.WebsocketPool.OnMessage = &onMessage
	miner.SubscribeToTopics()

	fmt.Println("Miner is running")
	fmt.Println(len(miner.WebsocketPool.connections), "websocket connections")

	for i := 0; ; i++ {
		time.Sleep(time.Minute)
		if miner.Options.RequiresStreamActivity() {
			if err := miner.UpdateStreamerTopicSubscriptions(); err != nil {
				fmt.Println("Error updating streamer topic subscriptions", err)
			}
		}

		if miner.Options.MineWatchtime {
			for _, user := range miner.Users {
				user.Chat.RevalidateChannelSubscriptions()
			}
		}

		if miner.Options.MinePoints {
			for _, user := range miner.Users {
				if err := miner.MinePoints(user); err != nil {
					fmt.Println("Error mining points", err)
				}
			}
		}

		// once an hour should be enough
		if i%60 == 59 {
			err := miner.UpdateVersions()
			if err != nil {
				fmt.Println("Error updating versions", err)
			}
		}
	}
}

func (miner *Miner) OnMessage(message WebsocketMessage) {
	// TODO: can we use a map here?
	switch message.Topics[0] {
	case "raid":
		switch message.Type {
		case "raid_update_v2":
			miner.OnRaidUpdate(message)
		}
	case "community-moments-channel-v1":
		switch message.Type {
		case "active":
			miner.OnMoment(message)
		}
	case "community-points-user-v1":
		switch message.Type {
		case "points-earned", "points-spent":
			miner.OnPointsUpdate(message)
		case "claim-available":
			miner.OnClaimAvailable(message)
		}
	case "predictions-channel-v1":
		switch message.Type {
		case "event-created":
			miner.OnPredictionUpdate(message)
		case "event-updated":
			miner.OnPredictionUpdate(message)
		}
	case "predictions-user-v1":
		switch message.Type {
		case "prediction-result":
		case "prediction-made":
		}
	case "video-playback-by-id":
		switch message.Type {
		case "stream-up":
			miner.OnStreamUp(message)
		case "stream-down":
			miner.OnStreamDown(message)
		case "viewcount":
			miner.OnViewcount(message)
		}
	}
}

func (miner *Miner) UpdateStreamerTopicSubscriptions() error {
	for _, streamer := range miner.Streamers {
		live := streamer.IsLive()
		if live != streamer.WasLive {
			if live {
				for _, topic := range streamer.LiveTopics {
					err := miner.WebsocketPool.ListenTopic(topic)
					if err != nil {
						fmt.Println("Error listening topic", err)
					}
				}
			} else {
				for _, topic := range streamer.LiveTopics {
					err := miner.WebsocketPool.UnlistenTopic(topic)
					if err != nil {
						fmt.Println("Error unlistening topic", err)
					}
				}
			}
			streamer.WasLive = live
		}
	}
	return nil
}

func (miner *Miner) UpdateVersions() error {
	response, err := miner.DefaultUser.GraphQL.Client.Get("https://twitch.tv")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	text, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	buildID := strings.Split(strings.Split(string(text), "__twilightBuildID=\"")[1], "\"")[0]
	for _, user := range miner.Users {
		user.GraphQL.ClientVersion = buildID
	}

	fmt.Println("Client version", buildID)

	regex := regexp.MustCompile(`https:\/\/[a-z-.]+\/config\/settings\.[^.]+\.js`)
	url := regex.FindString(string(text))
	response, err = miner.DefaultUser.GraphQL.Client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	text, err = io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	spadeUrl := strings.Split(strings.Split(string(text), "\"spade_url\":\"")[1], "\"")[0]
	miner.SpadeUrl = spadeUrl

	fmt.Println("Spade URL", spadeUrl)

	return nil
}

func NewMiner(options Options) *Miner {
	pool := NewWebsocketPool()
	state := LoadPersistentState(options)

	miner := &Miner{
		options,
		pool,
		nil,
		map[string]*User{},
		map[string]*Streamer{},
		map[string]*Prediction{},
		state,
		"",
		sync.Mutex{},
	}
	return miner
}
