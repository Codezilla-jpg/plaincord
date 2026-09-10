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
		return "Voice  ·  not in a call    → join   l leave   m mute"
	}
	state := "live"
	if muted {
		state = "muted"
	}
	return "Voice  ·  " + channelName + "  ·  " + state + "    m mute   l leave"
}

func Wave(frame int, speaking bool) string {
	bars := []string{"▁▂▃▄▅▆▇", "▂▃▅▇▆▄▂", "▃▅▇█▇▅▃", "▂▄▆▇▅▃▁", "▁▃▅▆▄▂▁"}
	if !speaking {
		return "▁▂▁▂▁▂▁"
	}
	if frame < 0 {
		frame = 0
	}
	return bars[frame%len(bars)]
}

func CallBanner(name, mic, headset string, muted, speaking bool, frame int) string {
	state := "live"
	if muted {
		state = "muted"
	}
	return "  🔊  " + name + "  ·  " + state + "  ·  mic " + mic + "  ·  headset " + headset + "  ·  " + Wave(frame, speaking && !muted)
}

func PeopleLine(name string, self, muted, speaking bool, frame int) string {
	mark := "○"
	if speaking && !muted {
		mark = "●"
	}
	label := name
	if self {
		label += "  (you)"
	}
	extra := ""
	if muted {
		extra = "  muted"
	} else if speaking {
		extra = "  " + Wave(frame, true)
	}
	return "  " + mark + "  " + label + extra
}
