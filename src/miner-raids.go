package miner

import (
	"encoding/json"
	"fmt"
)

func (miner *Miner) OnRaidUpdate(message WebsocketMessage) {
	var event raidEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		fmt.Println("Failed to unmarshal raid event", err)
		return
	}

	users := miner.GetUsersForStreamer(event.Raid.SourceID)

	for _, user := range users {
		if err := user.GraphQL.JoinRaid(event.Raid.ID); err != nil {
			fmt.Println("Failed to join raid:", err)
		}
	}
}

type raidEvent struct {
	Raid struct {
		ID       string `json:"id"`
		SourceID string `json:"source_id"`
	}
}
