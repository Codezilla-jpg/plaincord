package gateway

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Codezilla-jpg/plaincord/internal/model"
	"github.com/Codezilla-jpg/plaincord/internal/nav"
)

type Fake struct {
	Listener Listener
	mu       sync.Mutex
	guilds   []model.Guild
	channels map[string][]model.Channel
	friends  []model.Channel
	history  map[string][]model.ChatMessage
	people   map[string][]model.Participant
	nextID   int
	voiceCh  string
	voiceGID string
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
			{ID: "2", Name: "Arcade"},
		},
		friends: []model.Channel{
			{ID: "dm-ada", Name: "Ada", Kind: model.KindText, Position: 0},
			{ID: "call-ada", Name: "Call Ada", Kind: model.KindVoice, Position: 1},
			{ID: "dm-ben", Name: "Ben", Kind: model.KindText, Position: 2},
			{ID: "call-ben", Name: "Call Ben", Kind: model.KindVoice, Position: 3},
			{ID: "gcat", Name: "Groups", Kind: model.KindCategory, Position: 4},
			{ID: "dm-weekend", Name: "Weekend", Kind: model.KindText, CategoryID: "gcat", Position: 0},
			{ID: "call-weekend", Name: "Call Weekend", Kind: model.KindVoice, CategoryID: "gcat", Position: 1},
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
				{ID: "30", Name: "lobby", Kind: model.KindText, Position: 0},
				{ID: "31", Name: "clips", Kind: model.KindText, Position: 1},
				{ID: "32", Name: "Voice", Kind: model.KindCategory, Position: 2},
				{ID: "33", Name: "party", Kind: model.KindVoice, CategoryID: "32", Position: 0},
			},
		},
		history: map[string][]model.ChatMessage{
			"12": {
				{ID: "1", ChannelID: "12", Author: "Ada", Content: "hello", Timestamp: now},
				{ID: "2", ChannelID: "12", Author: "Ben", Content: "hi there", Timestamp: now},
			},
			"13": {{ID: "3", ChannelID: "13", Author: "Ada", Content: "off-topic lives here", Timestamp: now}},
			"10": {{ID: "4", ChannelID: "10", Author: "System", Content: "welcome to Home", Timestamp: now}},
			"30": {{ID: "6", ChannelID: "30", Author: "Cara", Content: "arcade lobby is open", Timestamp: now}},
			"31": {{ID: "7", ChannelID: "31", Author: "Ben", Content: "clip incoming", Timestamp: now}},
			"dm-ada": {
				{ID: "8", ChannelID: "dm-ada", Author: "Ada", Content: "got a minute?", Timestamp: now},
			},
			"dm-ben": {{ID: "9", ChannelID: "dm-ben", Author: "Ben", Content: "later tonight?", Timestamp: now}},
			"dm-weekend": {
				{ID: "10h", ChannelID: "dm-weekend", Author: "Cara", Content: "saturday 19:00", Timestamp: now},
				{ID: "11h", ChannelID: "dm-weekend", Author: "Ada", Content: "I'm in", Timestamp: now},
			},
		},
		people: map[string][]model.Participant{
			"15": {
				{ID: "you", Name: "you", Self: true},
				{ID: "ada", Name: "Ada", Speaking: true},
				{ID: "ben", Name: "Ben"},
			},
			"16": {
				{ID: "you", Name: "you", Self: true},
				{ID: "cara", Name: "Cara", Speaking: true},
			},
			"33": {
				{ID: "you", Name: "you", Self: true},
				{ID: "ben", Name: "Ben", Speaking: true},
				{ID: "cara", Name: "Cara"},
			},
			"call-ada": {
				{ID: "you", Name: "you", Self: true},
				{ID: "ada", Name: "Ada", Speaking: true},
			},
			"call-ben": {
				{ID: "you", Name: "you", Self: true},
				{ID: "ben", Name: "Ben", Speaking: true},
			},
			"call-weekend": {
				{ID: "you", Name: "you", Self: true},
				{ID: "ada", Name: "Ada", Speaking: true},
				{ID: "ben", Name: "Ben"},
				{ID: "cara", Name: "Cara"},
			},
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

func (f *Fake) Friends() []model.Channel {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]model.Channel, len(f.friends))
	copy(out, f.friends)
	return out
}

func (f *Fake) Channels(guildID string) []model.Channel {
	if guildID == nav.FriendsID {
		return f.Friends()
	}
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
	_ = name
	f.mu.Lock()
	defer f.mu.Unlock()
	f.voiceGID = guildID
	f.voiceCh = channelID
	return nil
}

func (f *Fake) LeaveVoice() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.voiceCh = ""
	f.voiceGID = ""
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

func (f *Fake) Participants(guildID, channelID string) []model.Participant {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.voiceCh == "" || f.voiceCh != channelID {
		return nil
	}
	_ = guildID
	src := f.people[channelID]
	out := make([]model.Participant, len(src))
	copy(out, src)
	return out
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
