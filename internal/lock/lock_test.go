package lock

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSecondAcquireIsBusyUntilRelease(t *testing.T) {
	name := fmt.Sprintf("pm-test-%d", time.Now().UnixNano())
	l, err := Acquire(name, "profil DEV")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Acquire(name, "profil VR")
	var busy *Busy
	if !errors.As(err, &busy) || !strings.Contains(busy.Owner, "profil DEV") {
		t.Fatalf("err = %v", err)
	}
	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
	l2, err := Acquire(name, "profil VR")
	if err != nil {
		t.Fatalf("lock non libéré : %v", err)
	}
	l2.Release()
}
