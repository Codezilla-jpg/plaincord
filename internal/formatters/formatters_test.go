package formatters

import (
	"strings"
	"testing"
	"time"
)

func TestMessageLine(t *testing.T) {
	moment := time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)
	line := MessageLine("Ada", "hello", moment)
	if !strings.Contains(line, "Ada: hello") {
		t.Fatalf("line %q", line)
	}
}

func TestChannelLabels(t *testing.T) {
	if ChannelLabel("text", "general") != "# general" {
		t.Fatal("text")
	}
	if !strings.HasSuffix(ChannelLabel("voice", "lounge"), "lounge") {
		t.Fatal("voice")
	}
	if ChannelLabel("category", "Text") != "TEXT" {
		t.Fatal("category")
	}
}

func TestVoiceBar(t *testing.T) {
	idle := VoiceBar(false, false, "")
	if !strings.Contains(idle, "not in a call") {
		t.Fatalf("%q", idle)
	}
	live := VoiceBar(true, false, "lounge")
	if !strings.Contains(live, "lounge") || !strings.Contains(live, "live") {
		t.Fatalf("%q", live)
	}
	muted := VoiceBar(true, true, "lounge")
	if !strings.Contains(muted, "muted") {
		t.Fatalf("%q", muted)
	}
}
