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
	"github.com/Codezilla-jpg/plaincord/internal/nav"
)

type Discord struct {
	Token     string
	Listener  Listener
	appID     string
	session   *discordgo.Session
	localCall bool
	localGID  string
	localCh   string
	localName string
	mu        sync.Mutex
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
	s := d.session
	gid, ch := d.localGID, d.localCh
	d.session = nil
	d.localCall = false
	d.localGID = ""
	d.localCh = ""
	d.localName = ""
	d.mu.Unlock()
	if s != nil && gid != "" && !skipVoicePresence(gid, ch) {
		_ = s.ChannelVoiceJoinManual(gid, "", false, false)
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

func (d *Discord) Friends() []model.Channel {
	s := d.sess()
	if s == nil || s.State == nil {
		return nil
	}
	chs := append([]*discordgo.Channel{}, s.State.PrivateChannels...)
	sort.Slice(chs, func(i, j int) bool {
		return dmName(chs[i]) < dmName(chs[j])
	})
	out := make([]model.Channel, 0, len(chs)*2)
	pos := 0
	for _, ch := range chs {
		if ch.Type != discordgo.ChannelTypeDM && ch.Type != discordgo.ChannelTypeGroupDM {
			continue
		}
		name := dmName(ch)
		out = append(out, model.Channel{ID: ch.ID, Name: name, Kind: model.KindText, Position: pos})
		pos++
		out = append(out, model.Channel{ID: "call:" + ch.ID, Name: "Call " + name, Kind: model.KindVoice, Position: pos})
		pos++
	}
	return out
}

func (d *Discord) Channels(guildID string) []model.Channel {
	if guildID == nav.FriendsID {
		return d.Friends()
	}
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

func skipVoicePresence(guildID, channelID string) bool {
	return guildID == nav.FriendsID || strings.HasPrefix(channelID, "call:")
}

func friendPeople(channelID, callName string) []model.Participant {
	out := []model.Participant{{ID: "you", Name: "you", Self: true}}
	name := strings.TrimPrefix(callName, "Call ")
	if name != "" && name != "you" {
		out = append(out, model.Participant{ID: channelID, Name: name, Speaking: true})
	}
	return out
}

func memberName(vs *discordgo.VoiceState) string {
	if vs == nil {
		return ""
	}
	if vs.Member != nil {
		if vs.Member.Nick != "" {
			return vs.Member.Nick
		}
		if vs.Member.User != nil && vs.Member.User.Username != "" {
			return vs.Member.User.Username
		}
	}
	return vs.UserID
}

func guildPeople(channelID, meID string, states []*discordgo.VoiceState) []model.Participant {
	out := []model.Participant{{ID: "you", Name: "you", Self: true}}
	for _, vs := range states {
		if vs == nil || vs.ChannelID != channelID {
			continue
		}
		if meID != "" && vs.UserID == meID {
			continue
		}
		out = append(out, model.Participant{
			ID:       vs.UserID,
			Name:     memberName(vs),
			Muted:    vs.SelfMute || vs.Mute,
			Speaking: true,
		})
	}
	return out
}

func (d *Discord) JoinVoice(guildID, channelID, name string) error {
	s := d.sess()
	if s == nil {
		return fmt.Errorf("not connected")
	}
	d.mu.Lock()
	prevGID, prevCh := d.localGID, d.localCh
	d.localCall = true
	d.localGID = guildID
	d.localCh = channelID
	d.localName = name
	d.mu.Unlock()
	if !skipVoicePresence(prevGID, prevCh) && prevGID != "" && prevGID != guildID {
		_ = s.ChannelVoiceJoinManual(prevGID, "", false, false)
	}
	if skipVoicePresence(guildID, channelID) {
		return nil
	}
	_ = s.ChannelVoiceJoinManual(guildID, channelID, false, false)
	return nil
}

func (d *Discord) LeaveVoice() error {
	d.mu.Lock()
	s := d.session
	gid, ch := d.localGID, d.localCh
	d.localCall = false
	d.localGID = ""
	d.localCh = ""
	d.localName = ""
	d.mu.Unlock()
	if s == nil || gid == "" || skipVoicePresence(gid, ch) {
		return nil
	}
	return s.ChannelVoiceJoinManual(gid, "", false, false)
}

func (d *Discord) SetMute(muted bool) error {
	d.mu.Lock()
	if !d.localCall {
		d.mu.Unlock()
		return fmt.Errorf("not in a call")
	}
	gid, ch := d.localGID, d.localCh
	d.mu.Unlock()
	if skipVoicePresence(gid, ch) {
		return nil
	}
	s := d.sess()
	if s == nil {
		return nil
	}
	return s.ChannelVoiceJoinManual(gid, ch, muted, false)
}

func (d *Discord) Participants(guildID, channelID string) []model.Participant {
	d.mu.Lock()
	active := d.localCall && d.localGID == guildID && d.localCh == channelID
	name := d.localName
	d.mu.Unlock()
	if !active {
		return nil
	}
	if skipVoicePresence(guildID, channelID) {
		return friendPeople(channelID, name)
	}
	return guildPeople(channelID, d.meID(), d.voiceStates(guildID))
}

func (d *Discord) meID() string {
	s := d.sess()
	if s == nil || s.State == nil || s.State.User == nil {
		return ""
	}
	return s.State.User.ID
}

func (d *Discord) voiceStates(guildID string) []*discordgo.VoiceState {
	s := d.sess()
	if s == nil || s.State == nil {
		return nil
	}
	guild, err := s.State.Guild(guildID)
	if err != nil || guild == nil {
		return nil
	}
	return guild.VoiceStates
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
	case discordgo.ChannelTypeGuildVoice, discordgo.ChannelTypeGuildStageVoice:
		return model.KindVoice, true
	default:
		return "", false
	}
}

func dmName(ch *discordgo.Channel) string {
	if ch == nil {
		return "dm"
	}
	if ch.Name != "" {
		return ch.Name
	}
	names := make([]string, 0, len(ch.Recipients))
	for _, u := range ch.Recipients {
		if u != nil && u.Username != "" {
			names = append(names, u.Username)
		}
	}
	if len(names) == 0 {
		return "dm"
	}
	return strings.Join(names, ", ")
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
