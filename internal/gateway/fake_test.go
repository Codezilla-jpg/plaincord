package gateway

import (
	"testing"

	"github.com/Codezilla-jpg/plaincord/internal/model"
)

type sinkListener struct {
	ready    bool
	messages int
}

func (s *sinkListener) OnReady()                    { s.ready = true }
func (s *sinkListener) OnMessage(model.ChatMessage) { s.messages++ }
func (s *sinkListener) OnError(string)              {}

func TestFakeGatewayFlow(t *testing.T) {
	var s sinkListener
	gw := NewFake(&s)
	if err := gw.Start(); err != nil {
		t.Fatal(err)
	}
	if !s.ready {
		t.Fatal("not ready")
	}
	guilds := gw.Guilds()
	if len(guilds) != 2 || guilds[0].Name != "Home" || guilds[1].Name != "Friends" {
		t.Fatalf("%+v", guilds)
	}
	channels := gw.Channels("1")
	var hasGeneral, hasLounge bool
	for _, ch := range channels {
		if ch.Name == "general" && ch.Kind == model.KindText {
			hasGeneral = true
		}
		if ch.Name == "lounge" && ch.Kind == model.KindVoice {
			hasLounge = true
		}
	}
	if !hasGeneral || !hasLounge {
		t.Fatalf("%+v", channels)
	}
	hist, err := gw.History("12", 80)
	if err != nil || len(hist) == 0 {
		t.Fatalf("history %v %v", hist, err)
	}
	msg, err := gw.Send("12", "ping")
	if err != nil || msg.Content != "ping" {
		t.Fatalf("send %v %v", msg, err)
	}
	if s.messages != 1 {
		t.Fatalf("messages %d", s.messages)
	}
	if err := gw.JoinVoice("1", "15", "lounge"); err != nil {
		t.Fatal(err)
	}
	if err := gw.SetMute(true); err != nil {
		t.Fatal(err)
	}
	if err := gw.LeaveVoice(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
}
