package miner

import "strings"

type Options struct {
	MinePoints           bool
	PrioritizeStreaks    bool
	ConcurrentPointLimit int
	ConcurrentWatchLimit int
	MiningStrategy       MiningStrategy
	StreamerPriority     map[string]int

	MineRaids   bool
	MineMoments bool

	MineWatchtime          bool
	WatchTimeOnlyLive      bool
	FollowChatSpam         bool
	FollowChatSpamChannels map[string]bool

	MinePredictions       bool
	PredictionsDataPoints int
	PredictionsMinPoints  int
	PredictionsMaxBet     int
	PredictionsMaxRatio   int
	PredictionsStealth    bool
	PredictionsStrategy   PredictionStrategy

	PersistentFile string
	DebugWebhook   string

	PrometheusEnabled bool
	PrometheusPort    int
	PrometheusHost    string
}

func (o Options) FollowChatSpamForChannel(channel string) bool {
	if enabled, ok := o.FollowChatSpamChannels[channel]; ok {
		return enabled
	}
	channel = strings.TrimPrefix(strings.ToLower(channel), "#")
	for configuredChannel, enabled := range o.FollowChatSpamChannels {
		if strings.TrimPrefix(strings.ToLower(configuredChannel), "#") == channel {
			return enabled
		}
	}

	return o.FollowChatSpam
}

func (o Options) RequiresStreamActivity() bool {
	return o.MinePoints || o.MineRaids || o.MineMoments || o.MinePredictions || (o.MineWatchtime && o.WatchTimeOnlyLive)
}
