package voice

import "testing"

func TestJoinMuteLeave(t *testing.T) {
	var c Controller
	if c.State.Connected {
		t.Fatal("start connected")
	}
	c.OnJoined("1", "15", "lounge")
	if !c.State.Connected || c.State.ChannelName != "lounge" || c.State.Muted {
		t.Fatalf("%+v", c.State)
	}
	muted, err := c.ToggleMute()
	if err != nil || !muted {
		t.Fatalf("mute %v %v", muted, err)
	}
	muted, err = c.ToggleMute()
	if err != nil || muted {
		t.Fatalf("unmute %v %v", muted, err)
	}
	c.OnLeft()
	if c.State.Connected || c.State.ChannelName != "" {
		t.Fatalf("%+v", c.State)
	}
}

func TestMuteWithoutCall(t *testing.T) {
	var c Controller
	if _, err := c.ToggleMute(); err != ErrNotInCall {
		t.Fatalf("got %v", err)
	}
}
