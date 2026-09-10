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

func TestWaveAndCallBanner(t *testing.T) {
	if Wave(0, false) == Wave(0, true) {
		t.Fatal("speaking wave should differ")
	}
	banner := CallBanner("lounge", "mic0", "head0", false, true, 1)
	if !strings.Contains(banner, "lounge") || !strings.Contains(banner, "mic0") || !strings.Contains(banner, "head0") {
		t.Fatalf("%q", banner)
	}
	line := PeopleLine("Ada", false, false, true, 2)
	if !strings.Contains(line, "Ada") || !strings.Contains(line, "●") {
		t.Fatalf("%q", line)
	}
}
