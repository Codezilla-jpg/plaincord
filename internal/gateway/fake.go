package gateway

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Codezilla-jpg/plaincord/internal/model"
)

type Fake struct {
	Listener Listener
	mu       sync.Mutex
	guilds   []model.Guild
	channels map[string][]model.Channel
	history  map[string][]model.ChatMessage
	nextID   int
	voiceCh  string
	appID    string
}

func NewFake(listener Listener) *Fake {
	if listener == nil {
		listener = NopListener{}
	}
	now := time.Now()
	return &Fake{
		Listener: listener,
		appID:    "0",
		nextID:   100,
		guilds: []model.Guild{
			{ID: "1", Name: "Home"},
			{ID: "2", Name: "Friends"},
		},
		channels: map[string][]model.Channel{
			"1": {
				{ID: "10", Name: "welcome", Kind: model.KindText, Position: 0},
				{ID: "11", Name: "Text", Kind: model.KindCategory, Position: 1},
				{ID: "12", Name: "general", Kind: model.KindText, CategoryID: "11", Position: 0},
				{ID: "13", Name: "random", Kind: model.KindText, CategoryID: "11", Position: 1},
				{ID: "14", Name: "Voice", Kind: model.KindCategory, Position: 2},
				{ID: "15", Name: "lounge", Kind: model.KindVoice, CategoryID: "14", Position: 0},
				{ID: "16", Name: "gaming", Kind: model.KindVoice, CategoryID: "14", Position: 1},
			},
			"2": {
				{ID: "20", Name: "chat", Kind: model.KindText, Position: 0},
				{ID: "21", Name: "call", Kind: model.KindVoice, Position: 1},
			},
		},
		history: map[string][]model.ChatMessage{
			"12": {
				{ID: "1", ChannelID: "12", Author: "Ada", Content: "hello", Timestamp: now},
				{ID: "2", ChannelID: "12", Author: "Ben", Content: "hi there", Timestamp: now},
			},
			"13": {{ID: "3", ChannelID: "13", Author: "Ada", Content: "off-topic lives here", Timestamp: now}},
			"10": {{ID: "4", ChannelID: "10", Author: "System", Content: "welcome to Home", Timestamp: now}},
			"20": {{ID: "5", ChannelID: "20", Author: "Cara", Content: "hey", Timestamp: now}},
		},
	}
}

func (f *Fake) Start() error {
	f.Listener.OnReady()
	return nil
}

func (f *Fake) Close() error { return nil }

func (f *Fake) ApplicationID() string { return f.appID }

func (f *Fake) Guilds() []model.Guild {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]model.Guild, len(f.guilds))
	copy(out, f.guilds)
	return out
}

func (f *Fake) Channels(guildID string) []model.Channel {
	f.mu.Lock()
	defer f.mu.Unlock()
	src := f.channels[guildID]
	out := make([]model.Channel, len(src))
	copy(out, src)
	return out
}

func (f *Fake) History(channelID string, limit int) ([]model.ChatMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	src := f.history[channelID]
	if limit <= 0 || limit > len(src) {
		limit = len(src)
	}
	out := make([]model.ChatMessage, limit)
	copy(out, src[len(src)-limit:])
	return out, nil
}

func (f *Fake) Send(channelID, content string) (model.ChatMessage, error) {
	f.mu.Lock()
	f.nextID++
	msg := model.ChatMessage{
		ID:        fmt.Sprintf("%d", f.nextID),
		ChannelID: channelID,
		Author:    "you",
		Content:   content,
		Timestamp: time.Now(),
	}
	f.history[channelID] = append(f.history[channelID], msg)
	f.mu.Unlock()
	f.Listener.OnMessage(msg)
	return msg, nil
}

func (f *Fake) JoinVoice(guildID, channelID, name string) error {
	_, _ = guildID, name
	f.mu.Lock()
	defer f.mu.Unlock()
	f.voiceCh = channelID
	return nil
}

func (f *Fake) LeaveVoice() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.voiceCh = ""
	return nil
}

func (f *Fake) SetMute(muted bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.voiceCh == "" {
		return fmt.Errorf("not in a call")
	}
	_ = muted
	return nil
}

func (f *Fake) JoinInvite(raw string) error {
	code := strings.TrimSpace(raw)
	if code == "" {
		return fmt.Errorf("invalid invite")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	id := fmt.Sprintf("%d", f.nextID)
	f.nextID++
	f.guilds = append(f.guilds, model.Guild{ID: id, Name: "invite-" + code})
	f.channels[id] = []model.Channel{
		{ID: id + "c", Name: "general", Kind: model.KindText},
	}
	return nil
}
