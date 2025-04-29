package miner

import "encoding/json"

type RawWebsocketMessage struct {
	Type  string                   `json:"type"`
	Data  *RawWebsocketMessageData `json:"data"`
	Error *string                  `json:"error"`
}
type RawWebsocketMessageData struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

type WebsocketMessageContent struct {
	Type string `json:"type"`
}

type WebsocketMessage struct {
	Topics []string
	Type   string
	Data   json.RawMessage
}
