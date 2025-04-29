package miner

type User struct {
	Username  string
	ID        string
	AuthToken string
	Chat      *Chat
	GraphQL   *GraphQL

	Streamers map[string]*Streamer
	Miner     *Miner
}

func (u *User) ConnectToChat() {
	if u.Chat == nil {
		u.Chat = NewChat(u)
	}
	go u.Chat.RunForever()
}

func NewUser(username string, authToken string) *User {
	user := &User{
		Username:  username,
		AuthToken: authToken,
		Streamers: map[string]*Streamer{},
	}
	user.GraphQL = NewGraphQL(user)
	return user
}
