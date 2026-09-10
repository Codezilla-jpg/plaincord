package gateway

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/Codezilla-jpg/plaincord/internal/auth"
	"github.com/Codezilla-jpg/plaincord/internal/invite"
	"github.com/Codezilla-jpg/plaincord/internal/model"
)

type Discord struct {
	Token    string
	Listener Listener
	appID    string
	session  *discordgo.Session
	voice    *discordgo.VoiceConnection
	mu       sync.Mutex
}

func NewDiscord(token string, listener Listener) *Discord {
	if listener == nil {
		listener = NopListener{}
	}
	return &Discord{Token: token, Listener: listener}
}

func (d *Discord) ApplicationID() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.appID
}

func (d *Discord) Start() error {
	token := strings.TrimPrefix(strings.TrimSpace(d.Token), "Bot ")
	s, err := discordgo.New(token)
	if err != nil {
		return err
	}
	s.Identify.Intents = discordgo.IntentsAll
	s.AddHandler(func(_ *discordgo.Session, r *discordgo.Ready) {
		id := ""
		if r.User != nil {
			id = r.User.ID
		}
		d.mu.Lock()
		d.appID = id
		d.mu.Unlock()
		if id != "" {
			_ = auth.SaveApplicationID(id)
		}
		d.Listener.OnReady()
	})
	s.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if m.GuildID == "" {
			return
		}
		d.Listener.OnMessage(toChat(m.Message))
	})
	if err := s.Open(); err != nil {
		d.Listener.OnError("Login failed — token rejected")
		return err
	}
	d.mu.Lock()
	d.session = s
	d.mu.Unlock()
	return nil
}

func (d *Discord) Close() error {
	d.mu.Lock()
	vc := d.voice
	s := d.session
	d.voice = nil
	d.session = nil
	d.mu.Unlock()
	if vc != nil {
		_ = vc.Disconnect()
	}
	if s != nil {
		return s.Close()
	}
	return nil
}

func (d *Discord) sess() *discordgo.Session {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.session
}

func (d *Discord) Guilds() []model.Guild {
	s := d.sess()
	if s == nil || s.State == nil {
		return nil
	}
	guilds := append([]*discordgo.Guild{}, s.State.Guilds...)
	sort.Slice(guilds, func(i, j int) bool {
		return guilds[i].Name < guilds[j].Name
	})
	out := make([]model.Guild, 0, len(guilds))
	for _, g := range guilds {
		out = append(out, model.Guild{ID: g.ID, Name: g.Name})
	}
	return out
}

func (d *Discord) Channels(guildID string) []model.Channel {
	s := d.sess()
	if s == nil {
		return nil
	}
	guild, err := s.State.Guild(guildID)
	if err != nil || guild == nil {
		return nil
	}
	var out []model.Channel
	for _, ch := range guild.Channels {
		kind, ok := kindOf(ch.Type)
		if !ok {
			continue
		}
		cat := ""
		if kind != model.KindCategory {
			cat = ch.ParentID
		}
		out = append(out, model.Channel{
			ID:         ch.ID,
			Name:       ch.Name,
			Kind:       kind,
			CategoryID: cat,
			Position:   ch.Position,
		})
	}
	return out
}

func (d *Discord) History(channelID string, limit int) ([]model.ChatMessage, error) {
	s := d.sess()
	if s == nil {
		return nil, fmt.Errorf("not connected")
	}
	if limit <= 0 {
		limit = 80
	}
	msgs, err := s.ChannelMessages(channelID, limit, "", "", "")
	if err != nil {
		return nil, err
	}
	out := make([]model.ChatMessage, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		out = append(out, toChat(msgs[i]))
	}
	return out, nil
}

func (d *Discord) Send(channelID, content string) (model.ChatMessage, error) {
	s := d.sess()
	if s == nil {
		return model.ChatMessage{}, fmt.Errorf("not connected")
	}
	msg, err := s.ChannelMessageSend(channelID, content)
	if err != nil {
		return model.ChatMessage{}, err
	}
	return toChat(msg), nil
}

func (d *Discord) JoinVoice(guildID, channelID, name string) error {
	_ = name
	s := d.sess()
	if s == nil {
		return fmt.Errorf("not connected")
	}
	d.mu.Lock()
	existing := d.voice
	d.mu.Unlock()
	if existing != nil && existing.GuildID == guildID {
		return existing.ChangeChannel(channelID, false, false)
	}
	if existing != nil {
		_ = existing.Disconnect()
	}
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.voice = vc
	d.mu.Unlock()
	return nil
}

func (d *Discord) LeaveVoice() error {
	d.mu.Lock()
	vc := d.voice
	d.voice = nil
	d.mu.Unlock()
	if vc == nil {
		return nil
	}
	return vc.Disconnect()
}

func (d *Discord) SetMute(muted bool) error {
	d.mu.Lock()
	vc := d.voice
	d.mu.Unlock()
	if vc == nil {
		return fmt.Errorf("not in a call")
	}
	return vc.ChangeChannel(vc.ChannelID, muted, false)
}

func (d *Discord) JoinInvite(raw string) error {
	code, err := invite.Parse(raw)
	if err != nil {
		return err
	}
	s := d.sess()
	if s == nil {
		return fmt.Errorf("not connected")
	}
	_, err = s.InviteAccept(code)
	return err
}

func kindOf(t discordgo.ChannelType) (model.Kind, bool) {
	switch t {
	case discordgo.ChannelTypeGuildCategory:
		return model.KindCategory, true
	case discordgo.ChannelTypeGuildText:
		return model.KindText, true
	case discordgo.ChannelTypeGuildVoice:
		return model.KindVoice, true
	default:
		return "", false
	}
}

func toChat(m *discordgo.Message) model.ChatMessage {
	author := "?"
	if m.Author != nil {
		author = m.Author.Username
		if m.Member != nil && m.Member.Nick != "" {
			author = m.Member.Nick
		}
	}
	stamp := m.Timestamp
	if stamp.IsZero() {
		stamp = time.Now()
	}
	return model.ChatMessage{
		ID:        m.ID,
		ChannelID: m.ChannelID,
		Author:    author,
		Content:   m.Content,
		Timestamp: stamp,
	}
}
