package nav

import (
	"github.com/Codezilla-jpg/plaincord/internal/chantree"
	"github.com/Codezilla-jpg/plaincord/internal/formatters"
	"github.com/Codezilla-jpg/plaincord/internal/model"
)

const FriendsID = "@friends"

type Column int

const (
	ColServers Column = iota
	ColChannels
	ColChat
)

type Input int

const (
	Up Input = iota
	Down
	Left
	Right
	Enter
)

type Action int

const (
	ActionNone Action = iota
	ActionPreview
	ActionFocusChannels
	ActionOpenChat
	ActionJoinCall
	ActionLeaveChat
	ActionLeaveChannels
)

type Server struct {
	ID      string
	Name    string
	Friends bool
}

type Row struct {
	Label string
	Cat   bool
	Ch    *model.Channel
}

type State struct {
	Column        Column
	ServerIdx     int
	ChannelIdx    int
	ChatOpen      bool
	InCall        bool
	CallGuildID   string
	CallChannelID string
	CallName      string
	OpenChannelID string
}

func Rail(guilds []model.Guild) []Server {
	out := make([]Server, 0, 1+len(guilds))
	out = append(out, Server{ID: FriendsID, Name: "Friends", Friends: true})
	for _, g := range guilds {
		out = append(out, Server{ID: g.ID, Name: g.Name})
	}
	return out
}

func RowsFrom(channels []model.Channel) []Row {
	return flatten(chantree.Build(channels))
}

func flatten(nodes []model.TreeNode) []Row {
	var out []Row
	for _, n := range nodes {
		if n.Kind == model.KindCategory {
			out = append(out, Row{Label: formatters.ChannelLabel("category", n.Name), Cat: true})
			out = append(out, flatten(n.Children)...)
			continue
		}
		if n.Channel == nil {
			continue
		}
		ch := *n.Channel
		out = append(out, Row{
			Label: formatters.ChannelLabel(string(ch.Kind), ch.Name),
			Ch:    &ch,
		})
	}
	return out
}

func FirstOpenable(rows []Row) int {
	return NextOpenable(rows, -1, 1)
}

func NextOpenable(rows []Row, from, dir int) int {
	if len(rows) == 0 || dir == 0 {
		return -1
	}
	n := len(rows)
	i := from
	for range n {
		i += dir
		if i < 0 {
			i = n - 1
		} else if i >= n {
			i = 0
		}
		if i == from {
			return from
		}
		if openable(rows, i) {
			return i
		}
	}
	return -1
}

func openable(rows []Row, i int) bool {
	if i < 0 || i >= len(rows) {
		return false
	}
	return !rows[i].Cat && rows[i].Ch != nil
}

func Handle(s State, in Input, nServers int, rows []Row) (State, Action) {
	if nServers < 1 {
		nServers = 1
	}
	switch s.Column {
	case ColChat:
		return handleChat(s, in)
	case ColChannels:
		return handleChannels(s, in, rows)
	default:
		return handleServers(s, in, nServers, rows)
	}
}

func handleServers(s State, in Input, nServers int, rows []Row) (State, Action) {
	switch in {
	case Up:
		s.ServerIdx = wrap(s.ServerIdx-1, nServers)
		s.ChannelIdx = FirstOpenable(rows)
		return s, ActionPreview
	case Down:
		s.ServerIdx = wrap(s.ServerIdx+1, nServers)
		s.ChannelIdx = FirstOpenable(rows)
		return s, ActionPreview
	case Right, Enter:
		idx := FirstOpenable(rows)
		if idx < 0 {
			return s, ActionNone
		}
		s.Column = ColChannels
		s.ChannelIdx = idx
		return s, ActionFocusChannels
	}
	return s, ActionNone
}

func handleChannels(s State, in Input, rows []Row) (State, Action) {
	switch in {
	case Up:
		idx := NextOpenable(rows, s.ChannelIdx, -1)
		if idx >= 0 {
			s.ChannelIdx = idx
		}
		return s, ActionNone
	case Down:
		idx := NextOpenable(rows, s.ChannelIdx, 1)
		if idx >= 0 {
			s.ChannelIdx = idx
		}
		return s, ActionNone
	case Left:
		s.Column = ColServers
		s.ChatOpen = false
		return s, ActionLeaveChannels
	case Right, Enter:
		return activate(s, rows)
	}
	return s, ActionNone
}

func handleChat(s State, in Input) (State, Action) {
	switch in {
	case Left:
		s.Column = ColChannels
		s.ChatOpen = false
		return s, ActionLeaveChat
	}
	return s, ActionNone
}

func activate(s State, rows []Row) (State, Action) {
	if !openable(rows, s.ChannelIdx) {
		idx := FirstOpenable(rows)
		if idx < 0 {
			return s, ActionNone
		}
		s.ChannelIdx = idx
	}
	ch := rows[s.ChannelIdx].Ch
	switch ch.Kind {
	case model.KindText:
		s.Column = ColChat
		s.ChatOpen = true
		s.OpenChannelID = ch.ID
		return s, ActionOpenChat
	case model.KindVoice:
		s.InCall = true
		s.CallChannelID = ch.ID
		s.CallName = ch.Name
		return s, ActionJoinCall
	}
	return s, ActionNone
}

func wrap(i, n int) int {
	if n <= 0 {
		return 0
	}
	return (i%n + n) % n
}
