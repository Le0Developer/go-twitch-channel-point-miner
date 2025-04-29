package miner

type WebsocketTopic struct {
	Topic      string
	User       *User
	Streamer   *Streamer
	AssignedTo *WebsocketConnection
}

func (topic *WebsocketTopic) GetTopicName() string {
	if topic.User != nil {
		return topic.Topic + "." + topic.User.ID
	}
	if topic.Streamer != nil {
		return topic.Topic + "." + topic.Streamer.ID
	}
	return topic.Topic
}
