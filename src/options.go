package miner

type Options struct {
	MinePoints           bool
	PrioritizeStreaks    bool
	ConcurrentPointLimit int
	ConcurrentWatchLimit int
	MiningStrategy       MiningStrategy
	StreamerPriority     map[string]int

	MineRaids   bool
	MineMoments bool

	MineWatchtime     bool
	WatchTimeOnlyLive bool
	FollowChatSpam    bool

	MinePredictions       bool
	PredictionsDataPoints int
	PredictionsMinPoints  int
	PredictionsMaxBet     int
	PredictionsMaxRatio   int
	PredictionsStealth    bool
	PredictionsStrategy   PredictionStrategy

	PersistentFile string
	DebugWebhook   string
}

func (o Options) RequiresStreamActivity() bool {
	return o.MinePoints || o.MineRaids || o.MineMoments || o.MinePredictions || (o.MineWatchtime && o.WatchTimeOnlyLive)
}
