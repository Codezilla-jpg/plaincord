package audio

import "testing"

func TestProbeHasNames(t *testing.T) {
	info := Probe()
	if info.Mic == "" || info.Headset == "" {
		t.Fatalf("%+v", info)
	}
}

func TestRMSSilence(t *testing.T) {
	if rms16(make([]byte, 64)) != 0 {
		t.Fatal("silence")
	}
}
