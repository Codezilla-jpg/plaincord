package gateway

import (
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/Codezilla-jpg/plaincord/internal/model"
	"github.com/Codezilla-jpg/plaincord/internal/nav"
)

func TestSkipVoicePresence(t *testing.T) {
	if !skipVoicePresence(nav.FriendsID, "dm1") {
		t.Fatal("friends")
	}
	if !skipVoicePresence("1", "call:dm1") {
		t.Fatal("dm call row")
	}
	if skipVoicePresence("1", "15") {
		t.Fatal("guild voice still sends presence")
	}
}

func TestFriendPeople(t *testing.T) {
	got := friendPeople("call:dm1", "Call Ada")
	if len(got) != 2 || !got[0].Self || got[1].Name != "Ada" {
		t.Fatalf("%+v", got)
	}
}

func TestGuildPeopleAlwaysIncludesYou(t *testing.T) {
	got := guildPeople("15", "me", nil)
	if len(got) != 1 || !got[0].Self || got[0].Name != "you" {
		t.Fatalf("%+v", got)
	}
	states := []*discordgo.VoiceState{
		{UserID: "me", ChannelID: "15"},
		{UserID: "ada", ChannelID: "15", Member: &discordgo.Member{Nick: "Ada"}},
		{UserID: "x", ChannelID: "other"},
	}
	got = guildPeople("15", "me", states)
	if len(got) != 2 || !got[0].Self || got[1].Name != "Ada" || got[1].Self {
		t.Fatalf("%+v", got)
	}
}

func TestKindOfIncludesStageVoice(t *testing.T) {
	k, ok := kindOf(discordgo.ChannelTypeGuildStageVoice)
	if !ok || k != model.KindVoice {
		t.Fatalf("stage %v %v", k, ok)
	}
	k, ok = kindOf(discordgo.ChannelTypeGuildVoice)
	if !ok || k != model.KindVoice {
		t.Fatalf("voice %v %v", k, ok)
	}
}

func TestJoinVoiceRequiresSession(t *testing.T) {
	d := NewDiscord("x", nil)
	if err := d.JoinVoice("1", "15", "lounge"); err == nil {
		t.Fatal("expected not connected")
	}
}
