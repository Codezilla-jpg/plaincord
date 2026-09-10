package nav

import (
	"testing"

	"github.com/Codezilla-jpg/plaincord/internal/model"
)

func TestRailPinsFriends(t *testing.T) {
	rail := Rail([]model.Guild{{ID: "1", Name: "Home"}, {ID: "2", Name: "Arcade"}})
	if len(rail) != 3 || !rail[0].Friends || rail[0].ID != FriendsID || rail[1].Name != "Home" {
		t.Fatalf("%+v", rail)
	}
}

func TestLiveServerSwitch(t *testing.T) {
	s := State{}
	s, act := Handle(s, Down, 3, nil)
	if act != ActionPreview || s.ServerIdx != 1 || s.Column != ColServers {
		t.Fatalf("down %+v %v", s, act)
	}
	s, act = Handle(s, Up, 3, nil)
	if act != ActionPreview || s.ServerIdx != 0 {
		t.Fatalf("up %+v %v", s, act)
	}
	s, act = Handle(s, Up, 3, nil)
	if s.ServerIdx != 2 {
		t.Fatalf("wrap %+v", s)
	}
}

func TestRightEntersFirstChannel(t *testing.T) {
	rows := RowsFrom([]model.Channel{
		{ID: "c", Name: "Text", Kind: model.KindCategory, Position: 0},
		{ID: "10", Name: "general", Kind: model.KindText, CategoryID: "c", Position: 0},
		{ID: "11", Name: "lounge", Kind: model.KindVoice, CategoryID: "c", Position: 1},
	})
	s, act := Handle(State{}, Right, 2, rows)
	if act != ActionFocusChannels || s.Column != ColChannels || s.ChannelIdx != 1 {
		t.Fatalf("enter %+v %v rows=%+v", s, act, rows)
	}
}

func TestChannelArrowsSkipCategories(t *testing.T) {
	rows := RowsFrom([]model.Channel{
		{ID: "c", Name: "Text", Kind: model.KindCategory, Position: 0},
		{ID: "10", Name: "general", Kind: model.KindText, CategoryID: "c", Position: 0},
		{ID: "11", Name: "random", Kind: model.KindText, CategoryID: "c", Position: 1},
		{ID: "v", Name: "Voice", Kind: model.KindCategory, Position: 1},
		{ID: "12", Name: "lounge", Kind: model.KindVoice, CategoryID: "v", Position: 0},
	})
	s := State{Column: ColChannels, ChannelIdx: FirstOpenable(rows)}
	s, _ = Handle(s, Down, 2, rows)
	if rows[s.ChannelIdx].Ch == nil || rows[s.ChannelIdx].Ch.Name != "random" {
		t.Fatalf("down %+v %+v", s, rows[s.ChannelIdx])
	}
	s, _ = Handle(s, Down, 2, rows)
	if rows[s.ChannelIdx].Ch.Name != "lounge" {
		t.Fatalf("skip cat %+v", rows[s.ChannelIdx])
	}
	s, _ = Handle(s, Up, 2, rows)
	if rows[s.ChannelIdx].Ch.Name != "random" {
		t.Fatalf("up %+v", rows[s.ChannelIdx])
	}
}

func TestRightOpensChatOrCall(t *testing.T) {
	rows := RowsFrom([]model.Channel{
		{ID: "10", Name: "general", Kind: model.KindText, Position: 0},
		{ID: "12", Name: "lounge", Kind: model.KindVoice, Position: 1},
	})
	s := State{Column: ColChannels, ChannelIdx: 0}
	s, act := Handle(s, Right, 2, rows)
	if act != ActionOpenChat || s.Column != ColChat || s.OpenChannelID != "10" {
		t.Fatalf("chat %+v %v", s, act)
	}
	s, act = Handle(s, Left, 2, rows)
	if act != ActionLeaveChat || s.Column != ColChannels {
		t.Fatalf("leave chat %+v %v", s, act)
	}
	s.ChannelIdx = 1
	s, act = Handle(s, Right, 2, rows)
	if act != ActionJoinCall || !s.InCall || s.CallName != "lounge" || s.Column != ColChannels {
		t.Fatalf("call %+v %v", s, act)
	}
}

func TestCallKeepsChatNavigation(t *testing.T) {
	rows := RowsFrom([]model.Channel{
		{ID: "10", Name: "general", Kind: model.KindText, Position: 0},
		{ID: "12", Name: "lounge", Kind: model.KindVoice, Position: 1},
	})
	s := State{Column: ColChannels, ChannelIdx: 1, InCall: true, CallName: "lounge"}
	s.ChannelIdx = 0
	s, act := Handle(s, Right, 2, rows)
	if act != ActionOpenChat || !s.InCall || s.Column != ColChat {
		t.Fatalf("chat during call %+v %v", s, act)
	}
	s, act = Handle(s, Left, 2, rows)
	if s.Column != ColChannels || !s.InCall {
		t.Fatalf("back %+v %v", s, act)
	}
	s, act = Handle(s, Left, 2, rows)
	if s.Column != ColServers || !s.InCall {
		t.Fatalf("servers during call %+v %v", s, act)
	}
}

func TestFriendsAlwaysChatRows(t *testing.T) {
	rows := RowsFrom([]model.Channel{
		{ID: "dm1", Name: "Ada", Kind: model.KindText, Position: 0},
		{ID: "call1", Name: "Call Ada", Kind: model.KindVoice, Position: 1},
	})
	s, act := Handle(State{ServerIdx: 0}, Right, 3, rows)
	if act != ActionFocusChannels || rows[s.ChannelIdx].Ch.Kind != model.KindText {
		t.Fatalf("friends first %+v %v", s, act)
	}
	s, act = Handle(s, Right, 3, rows)
	if act != ActionOpenChat || s.OpenChannelID != "dm1" {
		t.Fatalf("dm %+v %v", s, act)
	}
}
