package voice

import (
	"errors"

	"github.com/Codezilla-jpg/plaincord/internal/model"
)

var ErrNotInCall = errors.New("not in a call")

type Controller struct {
	State model.VoiceState
}

func (c *Controller) OnJoined(guildID, channelID, name string) model.VoiceState {
	c.State = model.VoiceState{
		Connected:   true,
		Muted:       false,
		GuildID:     guildID,
		ChannelID:   channelID,
		ChannelName: name,
	}
	return c.State
}

func (c *Controller) OnLeft() model.VoiceState {
	c.State = model.VoiceState{}
	return c.State
}

func (c *Controller) ToggleMute() (bool, error) {
	if !c.State.Connected {
		return false, ErrNotInCall
	}
	c.State.Muted = !c.State.Muted
	return c.State.Muted, nil
}
