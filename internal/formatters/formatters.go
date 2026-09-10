package formatters

import (
	"strings"
	"time"
)

func Clock(moment time.Time) string {
	return moment.In(time.Local).Format("15:04")
}

func MessageLine(author, content string, moment time.Time) string {
	body := strings.ReplaceAll(content, "\r\n", "\n")
	body = strings.Trim(body, "\n")
	if body == "" {
		body = "·"
	}
	return Clock(moment) + " " + author + ": " + body
}

func ChannelLabel(kind, name string) string {
	switch kind {
	case "voice":
		return "🔊 " + name
	case "category":
		return strings.ToUpper(name)
	default:
		return "# " + name
	}
}

func VoiceBar(connected, muted bool, channelName string) string {
	if !connected {
		return "Voice  ·  not in a call    j join   l leave   m mute"
	}
	state := "live"
	if muted {
		state = "muted"
	}
	return "Voice  ·  " + channelName + "  ·  " + state + "    m mute   l leave"
}
