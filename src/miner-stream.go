package miner

import (
	"encoding/json"
	"fmt"
	"time"
)

func (miner *Miner) OnStreamUp(message WebsocketMessage) {
	streamerID := message.Topics[1]
	streamer := miner.GetStreamerByID(streamerID)
	if streamer == nil {
		return
	}

	streamer.LastLivePing = time.Now()
	streamer.Viewers = 0
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
	var event viewcountEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		fmt.Println("Failed to unmarshal viewcount event", err)
		return
	}

	streamer.LastLivePing = time.Now()
	streamer.Viewers = event.Viewers
}

type viewcountEvent struct {
	Viewers int `json:"viewers"`
}
