package model

import "time"

type Kind string

const (
	KindCategory Kind = "category"
	KindText     Kind = "text"
	KindVoice    Kind = "voice"
)

type Guild struct {
	ID   string
	Name string
}

type Channel struct {
	ID         string
	Name       string
	Kind       Kind
	CategoryID string
	Position   int
}

type ChatMessage struct {
	ID        string
	ChannelID string
	Author    string
	Content   string
	Timestamp time.Time
}

type VoiceState struct {
	Connected   bool
	Muted       bool
	GuildID     string
	ChannelID   string
	ChannelName string
}

type TreeNode struct {
	ID       string
	Name     string
	Kind     Kind
	Children []TreeNode
	Channel  *Channel
}
