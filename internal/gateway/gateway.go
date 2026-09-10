package gateway

import (
	"github.com/Codezilla-jpg/plaincord/internal/model"
)

type Listener interface {
	OnReady()
	OnMessage(model.ChatMessage)
	OnError(string)
}

type Gateway interface {
	Start() error
	Close() error
	Guilds() []model.Guild
	Channels(guildID string) []model.Channel
	History(channelID string, limit int) ([]model.ChatMessage, error)
	Send(channelID, content string) (model.ChatMessage, error)
	JoinVoice(guildID, channelID, name string) error
	LeaveVoice() error
	SetMute(muted bool) error
	ApplicationID() string
}

type NopListener struct{}

func (NopListener) OnReady()                    {}
func (NopListener) OnMessage(model.ChatMessage) {}
func (NopListener) OnError(string)              {}
