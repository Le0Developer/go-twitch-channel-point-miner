package miner

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/irc.v4"
)

type Chat struct {
	user *User

	config         irc.ClientConfig
	client         *irc.Client
	ctx            context.Context
	cancel         context.CancelFunc
	isConnected    bool
	joinedChannels []string

	channelState map[string]*channelState
	chatLock     sync.Mutex
}

func (c *Chat) RunForever() {
	for c.ctx != nil {
		fmt.Println("Connecting to chat...")
		if err := c.connect(); err != nil {
			fmt.Println("Chat connection error:", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Chat) Stop() {
	c.cancel()
	c.ctx = nil
}

func (c *Chat) connect() error {
	c.isConnected = false
	c.joinedChannels = []string{}

	conn, err := net.Dial("tcp", "irc.chat.twitch.tv:6697")
	if err != nil {
		return err
	}

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName: "irc.chat.twitch.tv",
	})

	config := c.config
	config.Handler = irc.HandlerFunc(c.handler)
	c.client = irc.NewClient(tlsConn, config)

	c.client.CapRequest("twitch.tv/commands", true)
	c.client.CapRequest("twitch.tv/membership", true)
	c.client.CapRequest("twitch.tv/tags", true)

	return c.client.RunContext(c.ctx)
}

func (c *Chat) handler(client *irc.Client, message *irc.Message) {
	if message.Command != "PRIVMSG" && message.Command != "USERNOTICE" && message.Command != "JOIN" && message.Command != "PART" { // too spammy
		fmt.Println("[irc] Received message:", message)
	}

	switch message.Command {
	case irc.RPL_ENDOFMOTD:
		c.isConnected = true
		go c.onConnect()
	case "PRIVMSG":
		c.message(message)
	case "PING":
		c.ping(message)
		// case "CLEARCHAT":
		// 	c.clearChat(message)
	}
}

func (c *Chat) onConnect() {
	c.RevalidateChannelSubscriptions()
}

func (c *Chat) RevalidateChannelSubscriptions() {
	if !c.isConnected {
		return
	}
	channels := []string{}
	channelsToJoin := []string{}
	for _, streamer := range c.user.Streamers {
		if !streamer.IsLive() && c.user.Miner.Options.WatchTimeOnlyLive {
			continue
		}
		channel := streamer.ChannelName()
		channels = append(channels, channel)
		if !slices.Contains(c.joinedChannels, channel) {
			channelsToJoin = append(channelsToJoin, channel)
		}
	}

	channelsToLeave := []string{}
	for _, channel := range c.joinedChannels {
		if !slices.Contains(channels, channel) {
			channelsToLeave = append(channelsToLeave, channel)
		}
	}

	c.joinedChannels = channels
	if len(channelsToJoin) > 0 {
		fmt.Println("Joining channels:", channelsToJoin)
		c.joinChannels(channelsToJoin)
	}
	if len(channelsToLeave) > 0 {
		fmt.Println("Leaving channels:", channelsToLeave)
		c.leaveChannels(channelsToLeave)
	}
}

func (c *Chat) joinChannels(channels []string) {
	// we can join 20 channels per 10s
	for i := 0; i < len(channels); i += 20 {
		end := i + 20
		if end > len(channels) {
			end = len(channels)
		}

		message := &irc.Message{
			Command: "JOIN",
			Params:  []string{strings.Join(channels[i:end], ",")},
		}

		c.client.WriteMessage(message)

		time.Sleep(12 * time.Second)
	}
}

func (c *Chat) leaveChannels(channels []string) {
	c.client.WriteMessage(&irc.Message{
		Command: "PART",
		Params:  []string{strings.Join(channels, ",")},
	})
}

func (c *Chat) ping(message *irc.Message) {
	msg := message.Copy()
	msg.Command = "PONG"

	c.client.WriteMessage(msg)
}

func (c *Chat) message(message *irc.Message) {
	if !c.user.Miner.Options.FollowChatSpam {
		return
	}

	c.chatLock.Lock()
	defer c.chatLock.Unlock()

	content := message.Trailing()
	subscriberState := message.Tags["subscriber"]
	isSubscriber := subscriberState == "1"

	// either bot command or has a link
	if strings.ContainsAny(content, ".!+ ") || len(content) > 20 {
		return
	}

	channel := message.Params[0]

	ch, ok := c.channelState[channel]
	if !ok {
		ch = &channelState{
			lastMessageTime:      time.Time{},
			messageFrequency:     map[string]int{},
			messageSubCount:      map[string]int{},
			uniqueMessageSenders: map[string]map[string]struct{}{},
		}
		c.channelState[channel] = ch
	}

	ch.totalMessages++
	ch.messageFrequency[content]++
	if isSubscriber {
		ch.messageSubCount[content]++
	}
	uniqueSenders, ok := ch.uniqueMessageSenders[content]
	if !ok {
		uniqueSenders = map[string]struct{}{}
		ch.uniqueMessageSenders[content] = uniqueSenders
	}
	uniqueSenders[message.Prefix.Name] = struct{}{}

	if time.Since(ch.lastReset) > 15*time.Second {
		ch.reset()
	}

	if ch.messageSubCount[content] > 1 && len(uniqueSenders) > 3 && content != ch.lastMessage && time.Since(ch.lastMessageTime) > 10*time.Second && !isSubscriber {
		// follow the emote spam
		fmt.Println("Following spam:", content, "from", channel)
		ch.lastMessage = content
		ch.lastMessageTime = time.Now()
		ch.reset()
		chance := 0.6
		if ch.totalMessages > 30 {
			chance += float64(ch.totalMessages) / 900
			if chance > 0.95 {
				chance = 0.95
			}
		}
		if rand.Float64() > chance {
			go c.client.WriteMessage(&irc.Message{
				Command: "PRIVMSG",
				Params:  []string{channel, content},
			})
			go c.user.Miner.Alert(fmt.Sprintf("Followed spam: %s from %s", content, channel))
		}
	}
}

func (c *Chat) clearChat(message *irc.Message) {
	banDuration := message.Tags["ban-duration"]
	if banDuration != "" {
		duration, err := strconv.Atoi(banDuration)
		if err != nil {
			fmt.Println("Failed to parse ban duration:", err)
			return
		}

		// we dont care
		if duration < 300 {
			return
		}
		go c.user.Miner.Alert(fmt.Sprintf("o7 %s was banned from %s for %s", message.Params[1], message.Params[0], time.Duration(duration)*time.Second))
	} else {
		go c.user.Miner.Alert(fmt.Sprintf("o7 %s was banned from %s permanently", message.Params[1], message.Params[0]))
	}
}

func NewChat(u *User) *Chat {
	config := irc.ClientConfig{
		Nick: u.Username,
		Pass: fmt.Sprintf("oauth:%s", u.AuthToken),
		User: u.Username,
		Name: u.Username,
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Chat{
		user:           u,
		config:         config,
		client:         nil,
		ctx:            ctx,
		cancel:         cancel,
		isConnected:    false,
		joinedChannels: []string{},
		channelState:   map[string]*channelState{},
		chatLock:       sync.Mutex{},
	}
}

type channelState struct {
	lastMessageTime      time.Time
	lastMessage          string
	lastReset            time.Time
	messageFrequency     map[string]int
	messageSubCount      map[string]int
	uniqueMessageSenders map[string]map[string]struct{}
	totalMessages        int
}

func (c *channelState) reset() {
	c.messageFrequency = map[string]int{}
	c.messageSubCount = map[string]int{}
	c.uniqueMessageSenders = map[string]map[string]struct{}{}
	if c.totalMessages < 100 {
		c.totalMessages = 0
	} else {
		c.totalMessages -= 80
	}
	c.lastReset = time.Now()
}
