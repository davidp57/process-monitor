package affinity

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// The test binary doubles as a third-party process: with PM_HELPER=sleep
// it waits, hidden, until the test kills it by its PID.
func TestMain(m *testing.M) {
	if os.Getenv("PM_HELPER") == "sleep" {
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// copySelf gives the test binary another image name, to be found by name.
func copySelf(t *testing.T, name string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst := filepath.Join(t.TempDir(), name)
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, src); err != nil {
		t.Fatal(err)
	}
	out.Close()
	return dst
}

func startHelper(t *testing.T, exe string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "PM_HELPER=sleep")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	return cmd
}

func affinityOf(t *testing.T, pid int) Mask {
	t.Helper()
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	m, err := Get(h)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestKeeperHoldsNamedProcessesThenRestores(t *testing.T) {
	sys, err := System()
	if err != nil {
		t.Fatal(err)
	}
	if sys.Count() < 2 {
		t.Skip("one core: nothing to share")
	}
	target := Mask(1) // core 0 alone, which differs from the machine's mask
	name := "pmaff-other.exe"
	exe := copySelf(t, name)
	bystander := startHelper(t, copySelf(t, "pmaff-bystander.exe"))
	first := startHelper(t, exe)

	var events []string
	k := &Keeper{Mask: target, Names: []string{"PMAFF-OTHER.EXE"}, Event: func(m string) { events = append(events, m) }}
	k.Sweep()
	if got := affinityOf(t, first.Process.Pid); got != target {
		t.Fatalf("after a sweep: %s, want %s", got, target)
	}
	if got := affinityOf(t, bystander.Process.Pid); got != sys {
		t.Fatalf("a process not named was changed: %s", got)
	}
	if len(events) != 1 || !strings.Contains(events[0], name) {
		t.Fatalf("events = %q", events)
	}
	k.Sweep() // nothing to do: nothing more said
	if len(events) != 1 {
		t.Fatalf("a second sweep spoke again: %q", events)
	}

	// The process starts again: the next sweep catches the new one.
	_ = first.Process.Kill()
	_ = first.Wait()
	second := startHelper(t, exe)
	k.Sweep()
	if got := affinityOf(t, second.Process.Pid); got != target {
		t.Fatalf("restarted process: %s, want %s", got, target)
	}

	k.Restore()
	if got := affinityOf(t, second.Process.Pid); got != sys {
		t.Fatalf("after Restore: %s, want the original %s", got, sys)
	}
}

func TestRestoreLeavesAProcessChangedByOthers(t *testing.T) {
	sys, err := System()
	if err != nil {
		t.Fatal(err)
	}
	if sys.Count() < 3 {
		t.Skip("needs three cores")
	}
	cmd := startHelper(t, copySelf(t, "pmaff-moved.exe"))
	k := &Keeper{Mask: 1, Names: []string{"pmaff-moved.exe"}}
	k.Sweep()
	h, err := windows.OpenProcess(keeperAccess, false, uint32(cmd.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	if err := Set(h, 2); err != nil { // someone else moves it to core 1
		t.Fatal(err)
	}
	k.Restore()
	if got := affinityOf(t, cmd.Process.Pid); got != 2 {
		t.Fatalf("Restore overrode another change: %s", got)
	}
}
