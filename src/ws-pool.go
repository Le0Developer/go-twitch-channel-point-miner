package miner

import (
	"fmt"
	"slices"
)

const (
	websocketUrl           = "wss://pubsub-edge.twitch.tv"
	maxTopicsPerConnection = 50
)

type WebsocketPool struct {
	connections   []*WebsocketConnection
	connectionIDs int
	topics        []*WebsocketTopic
	OnMessage     *func(WebsocketMessage)
}

func (pool *WebsocketPool) ListenTopic(topic *WebsocketTopic) error {
	pool.topics = append(pool.topics, topic)
	return pool.SubmitTopic(topic)
}

func (pool *WebsocketPool) UnlistenTopic(topic *WebsocketTopic) error {
	index := slices.Index(pool.topics, topic)
	if index != -1 {
		pool.topics = append(pool.topics[:index], pool.topics[index+1:]...)
	} else {
		return fmt.Errorf("topic not found in pool")
	}
	return nil
}

func (pool *WebsocketPool) SubmitTopic(topic *WebsocketTopic) error {
	for _, conn := range pool.connections {
		if len(conn.topics) < maxTopicsPerConnection {
			return conn.ListenTopics(topic)
		}
	}
	fmt.Println("No connections available, creating a new one")
	// No connections available or at the limit, create a new one
	conn := NewWebsocketConnection(pool)
	pool.connections = append(pool.connections, conn)
	return conn.ListenTopics(topic)
}

func (pool *WebsocketPool) RevalidateTopics() error {
	// when a connection is closed, we just remove it from the pool
	// and this will pick up the missing topics
	missingTopics := []*WebsocketTopic{}
	for _, topic := range pool.topics {
		if topic.AssignedTo == nil || !slices.Contains(pool.connections, topic.AssignedTo) {
			missingTopics = append(missingTopics, topic)
		}
	}

	// TODO: this may send up to 50 INDIVIDUAL listen messages to the WS
	// we should probably batch them
	for _, topic := range missingTopics {
		pool.SubmitTopic(topic)
	}
	return nil
}

func (pool *WebsocketPool) OnDisconnect(conn *WebsocketConnection) {
	fmt.Println("Connection disconnected", conn)
	for i, c := range pool.connections {
		if c == conn {
			pool.connections = append(pool.connections[:i], pool.connections[i+1:]...)
			break
		}
	}
	pool.RevalidateTopics()
}

func NewWebsocketPool() *WebsocketPool {
	return &WebsocketPool{
		[]*WebsocketConnection{},
		0,
		[]*WebsocketTopic{},
		nil,
	}
}
