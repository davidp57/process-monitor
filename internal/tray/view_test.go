package tray

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/davidp57/process-monitor/internal/session"
)

func TestViewFollowsTheSession(t *testing.T) {
	cases := []struct {
		name    string
		st      session.State
		watch   bool
		failed  bool
		color   Color
		status  string
		pause   bool
		resume  bool
		restart bool
		stop    bool
		detach  bool
	}{
		{"starting", session.State{Status: session.Starting}, false, false, Orange, "démarrage", false, false, false, true, true},
		{"running run", session.State{Status: session.Running, PID: 42}, false, false, Green, "pid 42", false, false, false, true, true},
		{"running watch", session.State{Status: session.Running, PID: 42}, true, false, Green, "pid 42", true, false, true, true, true},
		{"paused", session.State{Status: session.Paused, PID: 42}, true, false, Grey, "en pause", false, true, true, true, true},
		{"restarting", session.State{Status: session.Restarting, Restarts: 2}, true, false, Orange, "relance", false, false, true, true, true},
		{"stopping", session.State{Status: session.Stopping}, false, false, Orange, "actions de fin", false, false, false, true, false},
		{"failed before launch", session.State{Status: session.Starting}, false, true, Red, "en erreur", false, false, false, true, true},
		{"finished ok", session.State{Status: session.Finished}, false, false, Green, "terminé", false, false, false, false, false},
		{"finished failed", session.State{Status: session.Finished}, false, true, Red, "terminé", false, false, false, false, false},
	}
	for _, c := range cases {
		v := ViewOf("dcs", "DEV", c.st, c.watch, c.failed)
		if v.Color != c.color || !strings.Contains(v.Status, c.status) {
			t.Errorf("%s : couleur %v, statut %q", c.name, v.Color, v.Status)
		}
		got := [5]bool{v.CanPause, v.CanResume, v.CanRestart, v.CanStop, v.CanDetach}
		want := [5]bool{c.pause, c.resume, c.restart, c.stop, c.detach}
		if got != want {
			t.Errorf("%s : menu %v, attendu %v (pause, reprise, relance, arrêt, détacher)", c.name, got, want)
		}
		if !strings.Contains(v.Tooltip, "dcs / DEV") || len(v.Tooltip) > 127 {
			t.Errorf("%s : infobulle %q", c.name, v.Tooltip)
		}
	}
}

func TestRestartCountdownInStatus(t *testing.T) {
	for _, secs := range []int{4, 37} { // two deadlines: a fixed text cannot pass both
		v := ViewOf("x", "", session.State{Status: session.Restarting, NextStart: time.Now().Add(time.Duration(secs)*time.Second + 300*time.Millisecond)}, true, false)
		if want := fmt.Sprintf("dans %ds", secs); !strings.Contains(v.Status, want) {
			t.Errorf("statut %q, attendu %q", v.Status, want)
		}
		if strings.Contains(v.Tooltip, " / ") {
			t.Errorf("pas de profil, pas de séparateur : %q", v.Tooltip)
		}
	}
}

// Detaching means something different at each moment; the confirmation
// says which, like the terminal interface does.
func TestDetachConfirmation(t *testing.T) {
	cases := map[session.Status]string{
		session.Starting:   "Rien ne sera lancé",
		session.Restarting: "seront exécutées",
		session.Running:    "PAS exécutées",
		session.Paused:     "PAS exécutées",
	}
	for st, want := range cases {
		if got := DetachQuestion(st); !strings.Contains(got, want) {
			t.Errorf("%s : %q", st, got)
		}
	}
}

func TestIconIsAValidIcoWithTheColor(t *testing.T) {
	for _, c := range []Color{Green, Orange, Grey, Red} {
		ico := Icon(c)
		if binary.LittleEndian.Uint16(ico[0:]) != 0 || binary.LittleEndian.Uint16(ico[2:]) != 1 || binary.LittleEndian.Uint16(ico[4:]) != 1 {
			t.Fatalf("%v : en-tête ICO invalide % x", c, ico[:6])
		}
		size := binary.LittleEndian.Uint32(ico[14:])
		offset := binary.LittleEndian.Uint32(ico[18:])
		if int(offset+size) != len(ico) {
			t.Fatalf("%v : taille annoncée %d+%d, réelle %d", c, offset, size, len(ico))
		}
		pixels := ico[offset+40:] // after BITMAPINFOHEADER, BGRA rows bottom-up
		px := func(x, y int) [4]byte {
			i := ((iconSize-1-y)*iconSize + x) * 4
			return [4]byte{pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]}
		}
		if corner := px(0, 0); corner[3] != 0 {
			t.Errorf("%v : le coin devrait être transparent, %v", c, corner)
		}
		want := map[Color][4]byte{ // BGRA, written out: not derived from rgb()
			Green:  {0x43, 0xa0, 0x2e, 255},
			Orange: {0x00, 0x8a, 0xe0, 255},
			Grey:   {0x8a, 0x8a, 0x8a, 255},
			Red:    {0x2c, 0x2c, 0xd0, 255},
		}[c]
		if center := px(iconSize/2, iconSize/2); center != want {
			t.Errorf("%v : centre %v, attendu BGRA %v", c, center, want)
		}
	}
}
