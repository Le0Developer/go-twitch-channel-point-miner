package miner

import "time"

type Streamer struct {
	Username string
	ID       string

	Points        map[*User]int
	GotPointsOnce map[*User]bool
	BroadcastID   string

	Viewers      int
	LastLivePing time.Time
	WasLive      bool

	LiveTopics []*WebsocketTopic
}

func (s *Streamer) IsLive() bool {
	return time.Since(s.LastLivePing) < 5*time.Minute
}

func (s Streamer) ChannelName() string {
	return "#" + s.Username
}
