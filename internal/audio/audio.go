package audio

import (
	"encoding/binary"
	"math"
	"os/exec"
	"strings"
	"sync/atomic"
)

type Info struct {
	Mic     string
	Headset string
}

func Probe() Info {
	info := Info{Mic: "none", Headset: "none"}
	if name, ok := pulseDefault("source"); ok {
		info.Mic = name
	} else if name, ok := alsaFirst("arecord"); ok {
		info.Mic = name
	}
	if name, ok := pulseDefault("sink"); ok {
		info.Headset = name
	} else if name, ok := alsaFirst("aplay"); ok {
		info.Headset = name
	}
	return info
}

func pulseDefault(kind string) (string, bool) {
	out, err := exec.Command("pactl", "get-default-"+kind).Output()
	if err != nil {
		return "", false
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", false
	}
	return name, true
}

func alsaFirst(bin string) (string, bool) {
	out, err := exec.Command(bin, "-l").Output()
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "card ") {
			return line, true
		}
	}
	return "", false
}

type Session struct {
	Info    Info
	onLevel func(float64)
	muted   atomic.Bool
	rec     *exec.Cmd
	play    *exec.Cmd
	closed  atomic.Bool
}

func Start(onLevel func(float64)) (*Session, error) {
	s := &Session{Info: Probe(), onLevel: onLevel}
	if rec := captureCmd(); rec != nil {
		stdout, err := rec.StdoutPipe()
		if err == nil && rec.Start() == nil {
			s.rec = rec
			go readRMS(stdout, s)
		}
	}
	if play := playbackCmd(); play != nil {
		stdin, err := play.StdinPipe()
		if err == nil && play.Start() == nil {
			s.play = play
			go func() {
				defer stdin.Close()
				buf := make([]byte, 1600)
				for !s.closed.Load() {
					if s.muted.Load() {
						for i := range buf {
							buf[i] = 0
						}
					} else {
						tone(buf, 0.04)
					}
					if _, err := stdin.Write(buf); err != nil {
						return
					}
				}
			}()
		}
	}
	return s, nil
}

func (s *Session) SetMuted(muted bool) { s.muted.Store(muted) }

func (s *Session) Close() {
	if s == nil || !s.closed.CompareAndSwap(false, true) {
		return
	}
	if s.rec != nil && s.rec.Process != nil {
		_ = s.rec.Process.Kill()
		_ = s.rec.Wait()
	}
	if s.play != nil && s.play.Process != nil {
		_ = s.play.Process.Kill()
		_ = s.play.Wait()
	}
}

func captureCmd() *exec.Cmd {
	if look("parec") {
		return exec.Command("parec", "--raw", "--format=s16le", "--rate=16000", "--channels=1")
	}
	if look("arecord") {
		return exec.Command("arecord", "-q", "-f", "S16_LE", "-r", "16000", "-c", "1", "-t", "raw")
	}
	return nil
}

func playbackCmd() *exec.Cmd {
	if look("paplay") {
		return exec.Command("paplay", "--raw", "--format=s16le", "--rate=16000", "--channels=1")
	}
	if look("aplay") {
		return exec.Command("aplay", "-q", "-f", "S16_LE", "-r", "16000", "-c", "1", "-t", "raw")
	}
	return nil
}

func look(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func readRMS(r interface{ Read([]byte) (int, error) }, s *Session) {
	buf := make([]byte, 3200)
	for !s.closed.Load() {
		n, err := r.Read(buf)
		if err != nil {
			return
		}
		if s.onLevel == nil || s.muted.Load() {
			continue
		}
		s.onLevel(rms16(buf[:n]))
	}
}

func rms16(b []byte) float64 {
	if len(b) < 2 {
		return 0
	}
	var sum float64
	n := 0
	for i := 0; i+1 < len(b); i += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(b[i : i+2])))
		sum += v * v
		n++
	}
	if n == 0 {
		return 0
	}
	return math.Sqrt(sum/float64(n)) / 32768
}

func tone(buf []byte, amp float64) {
	const rate = 16000
	const freq = 220
	for i := 0; i+1 < len(buf); i += 2 {
		t := float64(i/2) / rate
		v := int16(amp * 32767 * math.Sin(2*math.Pi*freq*t))
		binary.LittleEndian.PutUint16(buf[i:i+2], uint16(v))
	}
}
