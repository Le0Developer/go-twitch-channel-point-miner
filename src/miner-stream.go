package miner

import (
	"time"
)

func (miner *Miner) OnStreamUp(message WebsocketMessage) {
	streamerID := message.Topics[1]
	streamer := miner.GetStreamerByID(streamerID)
	if streamer == nil {
		return
	}

	streamer.LastLivePing = time.Now()
	for u := range streamer.GotPointsOnce {
		delete(streamer.GotPointsOnce, u)
	}
}

func (miner *Miner) OnStreamDown(message WebsocketMessage) {
	streamerID := message.Topics[1]
	streamer := miner.GetStreamerByID(streamerID)
	if streamer == nil {
		return
	}

	streamer.LastLivePing = time.Time{}
	streamer.BroadcastID = ""
}

func (miner *Miner) OnViewcount(message WebsocketMessage) {
	streamerID := message.Topics[1]
	streamer := miner.GetStreamerByID(streamerID)
	if streamer == nil {
		return
	}
	streamer.LastLivePing = time.Now()
}
