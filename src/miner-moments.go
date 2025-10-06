package miner

import (
	"encoding/json"
	"fmt"
)

func (miner *Miner) OnMoment(message WebsocketMessage) {
	var event MomentActiveEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		fmt.Println("Failed to unmarshal moment event", err)
		return
	}

	streamerID := message.Topics[1]

	users := miner.GetUsersForStreamer(streamerID)
	for _, user := range users {
		if err := user.GraphQL.ClaimMoment(event.Data.MomentID); err != nil {
			fmt.Println("Failed to claim moment:", err)
		}
	}
}

type MomentActiveEvent struct {
	Data struct {
		MomentID string `json:"moment_id"`
	} `json:"data"`
}
