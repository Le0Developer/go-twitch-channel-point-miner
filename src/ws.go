package miner

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/net/websocket"
)

type WebsocketConnection struct {
	ID int

	conn        *websocket.Conn
	topics      []*WebsocketTopic
	pool        *WebsocketPool
	lastPing    time.Time
	lastPong    time.Time
	lastMessage time.Time
}

func (ws *WebsocketConnection) Connect() error {
	conn, err := websocket.Dial(websocketUrl, "", "https://www.twitch.tv")
	if err != nil {
		return err
	}
	ws.conn = conn
	go ws.HandleMessages()
	go ws.HandleKeepalive()
	return nil
}

func (ws *WebsocketConnection) HandleMessages() {
	for ws.conn != nil {
		var data RawWebsocketMessage
		if err := ws.readMessage(&data); err != nil {
			fmt.Println("Error reading message", err)
			break
		}

		if data.Type == "PONG" {
			ws.log("Received PONG in", time.Since(ws.lastPing))
			ws.lastPong = time.Now()
		} else if data.Type == "RECONNECT" {
			ws.log("Received signal to reconnect")
			break
		} else if data.Type == "RESPONSE" {
			if data.Error != nil && *data.Error != "" {
				ws.log("Received error", *data.Error)
				if strings.Contains(*data.Error, "ERR_BADAUTH") {
					ws.log("Received ERR_BADAUTH, disconnecting")
					break
				}
			}
		} else if data.Type == "MESSAGE" {
			ws.lastMessage = time.Now()
			ws.log("Received message", data.Data.Topic, data.Data.Message)
			var content WebsocketMessageContent
			if err := json.Unmarshal([]byte(data.Data.Message), &content); err != nil {
				fmt.Println("Error unmarshalling message content", err)
				break
			}
			topics := strings.Split(data.Data.Topic, ".")
			go (*ws.pool.OnMessage)(WebsocketMessage{topics, content.Type, json.RawMessage(data.Data.Message)})
		} else {
			spew.Dump(data)
		}
	}
	ws.Disconnect()
}

func (ws *WebsocketConnection) readMessage(data any) error {
	fullBuffer := []byte{}
	wasPartial := false
	// yes this is needed. twitch for some reason sends their larger events in multiple frames
	// and the websocket library does not handle this for us
	// even when using their websocket.JSON Codec
	// so we need to stitch the frames together ourselves until we get a valid json object
	//
	// for example a 8kb prediction event is sent in 3 frames
	for ws.conn != nil {
		var message []byte
		if err := websocket.Message.Receive(ws.conn, &message); err != nil {
			fmt.Println("Error reading message", err)
			return err
		}

		fullBuffer = append(fullBuffer, message...)

		if err := json.Unmarshal(fullBuffer, &data); err == nil {
			if wasPartial {
				ws.log("Received multiframe message", len(fullBuffer))
			}
			return nil
		}
		wasPartial = true
	}
	return fmt.Errorf("connection is terminated")
}

func (ws *WebsocketConnection) Disconnect() {
	ws.log("Disconnecting from websocket")
	if ws.conn != nil {
		_ = ws.conn.WriteClose(1000)
		ws.conn = nil
	}
	if err := ws.pool.OnDisconnect(ws); err != nil {
		ws.log("Error handling disconnect in pool", err)
	}
}

func (ws *WebsocketConnection) HandleKeepalive() {
	// >To keep the server from closing the connection, clients must send a PING command at least once every 5 minutes

	// pre-enconded PING message
	event := map[string]any{
		"type": "PING",
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return
	}

	for ws.conn != nil {
		ws.log("Sending PING")
		ws.lastPing = time.Now()

		_, err = ws.conn.Write(encoded)
		if err != nil {
			fmt.Println("Error sending PING", err)
			ws.Disconnect()
			return
		}

		//> If a client does not receive a PONG message within 10 seconds of issuing a PING command, it should reconnect to the server
		time.Sleep(10 * time.Second)
		if ws.lastPong.Before(ws.lastPing) {
			ws.log("Did not receive PONG, disconnecting")
			ws.Disconnect()
			break
		}

		// send next PING in 2 minutes + 0-20 seconds
		//> If a client uses timers to issue PING commands, it should add a small random jitter to the timer.
		time.Sleep(2*time.Minute + time.Duration(rand.Intn(20))*time.Second)
	}
}

func (ws *WebsocketConnection) ListenTopics(topics ...*WebsocketTopic) error {
	users := map[*User][]string{}
	for _, topic := range topics {
		ws.topics = append(ws.topics, topic)
		topic.AssignedTo = ws
		users[topic.User] = append(users[topic.User], topic.GetTopicName())
	}

	if ws.conn == nil {
		err := ws.Connect()
		if err != nil {
			return err
		}
	}

	for user, userTopics := range users {
		event := map[string]any{
			"type": "LISTEN",
			"data": map[string]any{
				"topics": userTopics,
			},
			"nonce": createRandomString(30),
		}
		if user != nil {
			event["data"].(map[string]any)["auth_token"] = user.AuthToken
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			return err
		}
		ws.log("Sending LISTEN", string(encoded))
		_, err = ws.conn.Write(encoded)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ws *WebsocketConnection) log(content ...any) {
	content = append([]any{fmt.Sprintf("[ws-%d]", ws.ID)}, content...)
	fmt.Println(content...)
}

func NewWebsocketConnection(pool *WebsocketPool) *WebsocketConnection {
	id := pool.connectionIDs
	pool.connectionIDs++
	return &WebsocketConnection{id, nil, []*WebsocketTopic{}, pool, time.Time{}, time.Time{}, time.Time{}}
}
