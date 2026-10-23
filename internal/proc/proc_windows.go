package proc

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ntdll            = windows.NewLazySystemDLL("ntdll.dll")
	ntSuspendProcess = ntdll.NewProc("NtSuspendProcess")
	ntResumeProcess  = ntdll.NewProc("NtResumeProcess")
)

const (
	jobObjectBasicAccountingInformation = 1
	jobObjectBasicProcessIDList         = 3
	jobObjectAssociateCompletionPort    = 7
	jobObjectExtendedLimitInformation   = 9

	jobMsgActiveProcessZero = 4
	jobMsgNewProcess        = 6

	processAccess = windows.SYNCHRONIZE | windows.PROCESS_QUERY_LIMITED_INFORMATION |
		windows.PROCESS_TERMINATE | windows.PROCESS_SUSPEND_RESUME | windows.PROCESS_SET_QUOTA
	stillActive = 259

	// handoverGrace is how long, after the tracked process exits, a process
	// of the same name may appear and take over before the session ends.
	handoverGrace = 3 * time.Second
)

type jobAssociateCompletionPort struct {
	CompletionKey  uintptr
	CompletionPort windows.Handle
}

type jobBasicAccounting struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

type winProcess struct {
	spec    Spec
	started time.Time
	cmd     *exec.Cmd
	mainH   windows.Handle
	job     windows.Handle // 0 when the program could not be put in a job
	port    windows.Handle

	mu       sync.Mutex
	tracked  windows.Handle
	trackPID int
	followed map[int]bool // PIDs already tracked; owned by follow()
	pumps    sync.WaitGroup

	mainExited chan struct{}
	mainCode   int // valid once mainExited is closed
	done       chan struct{}
	result     Result

	stop       chan struct{} // closed by Close: follow must return
	followDone chan struct{} // closed when follow has returned
	closeOnce  sync.Once
}

// Start launches spec.Command suspended, places it in a fresh job, then resumes it.
func Start(spec Spec) (Process, error) {
	p := &winProcess{spec: spec, followed: map[int]bool{}, mainExited: make(chan struct{}), done: make(chan struct{}),
		stop: make(chan struct{}), followDone: make(chan struct{})}
	if err := p.createJob(); err != nil {
		spec.event(fmt.Sprintf("Job Object indisponible (%v), suivi du seul process lancé", err))
		p.closeJob()
	}

	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	flags := uint32(windows.CREATE_SUSPENDED | windows.CREATE_NEW_PROCESS_GROUP)
	var readers []*os.File
	fail := func(err error) (Process, error) {
		for _, r := range readers {
			r.Close()
		}
		if p.mainH != 0 {
			windows.CloseHandle(p.mainH)
		}
		p.closeJob()
		return nil, err
	}
	if spec.Capture {
		flags |= windows.CREATE_NO_WINDOW
		for _, dst := range []*io.Writer{&cmd.Stdout, &cmd.Stderr} {
			r, w, err := os.Pipe()
			if err != nil {
				return fail(err)
			}
			readers = append(readers, r)
			*dst = w
			defer w.Close() // the child has its copy once started
		}
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags}
	if err := cmd.Start(); err != nil {
		return fail(err)
	}
	p.cmd = cmd
	p.started = time.Now()

	h, err := windows.OpenProcess(processAccess, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = cmd.Process.Kill()
		return fail(fmt.Errorf("ouverture du process %d : %w", cmd.Process.Pid, err))
	}
	p.mainH = h
	if p.job != 0 {
		if err := windows.AssignProcessToJobObject(p.job, h); err != nil {
			spec.event(fmt.Sprintf("rattachement au Job Object impossible (%v), suivi du seul process lancé", err))
			p.closeJob()
		}
	}
	if r, _, err := ntResumeProcess.Call(uintptr(h)); r != 0 {
		_ = cmd.Process.Kill()
		return fail(fmt.Errorf("reprise du process suspendu : %w", err))
	}

	if spec.Output != nil {
		for _, r := range readers {
			p.pumps.Add(1)
			go func() {
				defer p.pumps.Done()
				pump(r, spec.Output)
			}()
		}
	}
	go func() {
		_ = cmd.Wait()
		p.mainCode = cmd.ProcessState.ExitCode()
		close(p.mainExited)
	}()
	go p.follow()
	return p, nil
}

func (p *winProcess) createJob() error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	p.job = job
	// Allow breakaway: a launcher that asks for it must not fail to start
	// the program. A process that leaves the job is still found by name.
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK
	if _, err := windows.SetInformationJobObject(job, jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return err
	}
	port, err := windows.CreateIoCompletionPort(windows.InvalidHandle, 0, 0, 1)
	if err != nil {
		return err
	}
	p.port = port
	assoc := jobAssociateCompletionPort{CompletionKey: uintptr(job), CompletionPort: port}
	_, err = windows.SetInformationJobObject(job, jobObjectAssociateCompletionPort,
		uintptr(unsafe.Pointer(&assoc)), uint32(unsafe.Sizeof(assoc)))
	return err
}

func (p *winProcess) closeJob() {
	if p.port != 0 {
		windows.CloseHandle(p.port)
		p.port = 0
	}
	if p.job != 0 {
		windows.CloseHandle(p.job)
		p.job = 0
	}
}

// follow waits for the end of the session. Job notifications wake it up
// immediately; a one-second tick covers lost notifications and the search
// for a tracked process that left the job.
func (p *winProcess) follow() {
	defer close(p.followDone)
	var lostSince, lastAt time.Time
	var last *Result // the tracked process that exited, while looking for a successor
	lingerTold := false
	mainSeen := false
	for {
		select {
		case <-p.stop:
			return
		default:
		}
		wait := uint32(1000)
		if last != nil {
			wait = 250 // a successor shows up quickly; check often
		}
		if p.port != 0 {
			var code uint32
			var key uintptr
			var ov uintptr // job notifications carry a PID where an OVERLAPPED pointer would be
			err := windows.GetQueuedCompletionStatus(p.port, &code, &key, (**windows.Overlapped)(unsafe.Pointer(&ov)), wait)
			if err == nil && code == jobMsgNewProcess && p.spec.Track != "" {
				p.consider(int(ov))
			}
		} else if mainSeen {
			time.Sleep(time.Second)
		} else {
			select {
			case <-p.mainExited:
				mainSeen = true
			case <-time.After(time.Second):
			}
		}

		if p.spec.Track == "" {
			if p.allExited() {
				<-p.mainExited
				p.finish(Result{PID: p.cmd.Process.Pid, ExitCode: p.mainCode})
				return
			}
			continue
		}

		if p.trackedHandle() == 0 {
			p.searchTracked()
			if p.trackedHandle() != 0 && last != nil {
				p.spec.event(fmt.Sprintf("relais : %s (pid %d) a pris la suite du pid %d", p.spec.Track, p.PID(), last.PID))
				last = nil
			}
		}
		if h := p.trackedHandle(); h != 0 {
			if ev, _ := windows.WaitForSingleObject(h, 0); ev == windows.WAIT_OBJECT_0 {
				// Not the end yet: it may have handed over to a process of
				// the same name (a launcher named like its target).
				var code uint32
				_ = windows.GetExitCodeProcess(h, &code)
				last = &Result{PID: p.PID(), ExitCode: int(code)}
				lastAt = time.Now()
				p.mu.Lock()
				windows.CloseHandle(p.tracked)
				p.tracked = 0
				p.mu.Unlock()
				continue
			}
			continue
		}
		if last != nil {
			if time.Since(lastAt) <= handoverGrace {
				continue
			}
			// The after hooks must never run while anything of the session
			// still runs (a launcher about to start the game again, say).
			if !p.allExited() {
				if !lingerTold {
					lingerTold = true
					p.spec.event(fmt.Sprintf("%s s'est terminé, mais des process de la session tournent encore (pid %v) : attente de leur fin", p.spec.Track, p.jobPIDs()))
				}
				continue
			}
			p.finish(*last)
			return
		}
		if p.allExited() {
			if lostSince.IsZero() {
				lostSince = time.Now()
			}
			if time.Since(lostSince) > p.spec.TrackGrace {
				<-p.mainExited
				p.finish(Result{PID: p.cmd.Process.Pid, ExitCode: p.mainCode,
					Err: fmt.Errorf("%s n'est jamais apparu (le lanceur s'est terminé avec le code %d)", p.spec.Track, p.mainCode)})
				return
			}
		}
	}
}

func (p *winProcess) finish(r Result) {
	// Let the last output lines through before the end is announced. A
	// grandchild holding the pipes open must not delay it for long.
	drained := make(chan struct{})
	go func() { p.pumps.Wait(); close(drained) }()
	select {
	case <-drained:
	case <-time.After(time.Second):
	}
	p.result = r
	close(p.done)
}

// allExited reports whether nothing started by the session is still running.
func (p *winProcess) allExited() bool {
	if p.job == 0 {
		select {
		case <-p.mainExited:
			return true
		default:
			return false
		}
	}
	var info jobBasicAccounting
	if err := windows.QueryInformationJobObject(p.job, jobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil); err != nil {
		return false
	}
	return info.ActiveProcesses == 0
}

// searchTracked looks for the tracked process in the job, then system-wide
// among processes created after the launch (it may have left the job).
func (p *winProcess) searchTracked() {
	for _, pid := range p.jobPIDs() {
		if p.consider(pid) {
			return
		}
	}
	pids, err := FindByName(p.spec.Track)
	if err != nil {
		return
	}
	for _, pid := range pids {
		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
		if err != nil {
			continue
		}
		created := creationTime(h)
		windows.CloseHandle(h)
		if !created.Before(p.started.Add(-time.Second)) && p.consider(pid) {
			p.spec.event(fmt.Sprintf("%s (pid %d) a quitté le Job Object, suivi par son nom", p.spec.Track, pid))
			return
		}
	}
}

// consider adopts pid as the tracked process if its name matches.
func (p *winProcess) consider(pid int) bool {
	if p.trackedHandle() != 0 || pid == 0 {
		return p.trackedHandle() != 0
	}
	if p.followed[pid] {
		return false // already followed and exited: not a successor
	}
	h, err := windows.OpenProcess(processAccess, false, uint32(pid))
	if err != nil {
		return false
	}
	if !strings.EqualFold(imageName(h), p.spec.Track) {
		windows.CloseHandle(h)
		return false
	}
	p.mu.Lock()
	p.tracked, p.trackPID = h, pid
	p.mu.Unlock()
	p.followed[pid] = true
	p.spec.event(fmt.Sprintf("process suivi : %s (pid %d)", p.spec.Track, pid))
	return true
}

func (p *winProcess) trackedHandle() windows.Handle {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tracked
}

func (p *winProcess) jobPIDs() []int {
	if p.job == 0 {
		return nil
	}
	// JOBOBJECT_BASIC_PROCESS_ID_LIST: two DWORD counters (8 bytes), then
	// ULONG_PTR ids, which start at byte 8 on 32- and 64-bit alike.
	const max = 1024
	hdr := 8 / int(unsafe.Sizeof(uintptr(0)))
	buf := make([]uintptr, hdr+max)
	if err := windows.QueryInformationJobObject(p.job, jobObjectBasicProcessIDList,
		uintptr(unsafe.Pointer(&buf[0])), uint32(len(buf))*uint32(unsafe.Sizeof(buf[0])), nil); err != nil {
		return nil
	}
	n := int(*(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(&buf[0])) + 4)))
	pids := make([]int, 0, n)
	for i := 0; i < n && i < max; i++ {
		pids = append(pids, int(buf[hdr+i]))
	}
	return pids
}

func (p *winProcess) PID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.trackPID != 0 {
		return p.trackPID
	}
	return p.cmd.Process.Pid
}

func (p *winProcess) Done() <-chan struct{} { return p.done }
func (p *winProcess) Result() Result        { return p.result }

func (p *winProcess) Kill() error {
	var errs []error
	if p.job != 0 {
		errs = append(errs, windows.TerminateJobObject(p.job, 1))
	} else {
		errs = append(errs, windows.TerminateProcess(p.mainH, 1))
	}
	// Hold the lock: follow() may close this handle at any moment.
	p.mu.Lock()
	if h := p.tracked; h != 0 {
		if ev, _ := windows.WaitForSingleObject(h, 0); ev != windows.WAIT_OBJECT_0 {
			errs = append(errs, windows.TerminateProcess(h, 1))
		}
	}
	p.mu.Unlock()
	return errors.Join(errs...)
}

// handles returns every process of the session: the job members plus the
// tracked process. The caller closes them.
func (p *winProcess) handles() []windows.Handle {
	seen := map[int]bool{}
	var hs []windows.Handle
	for _, pid := range append(p.jobPIDs(), p.PID()) {
		if seen[pid] {
			continue
		}
		seen[pid] = true
		if h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, uint32(pid)); err == nil {
			hs = append(hs, h)
		}
	}
	return hs
}

func (p *winProcess) each(fn *windows.LazyProc) error {
	hs := p.handles()
	if len(hs) == 0 {
		return errors.New("aucun process à atteindre")
	}
	var errs []error
	for _, h := range hs {
		if r, _, err := fn.Call(uintptr(h)); r != 0 {
			errs = append(errs, err)
		}
		windows.CloseHandle(h)
	}
	return errors.Join(errs...)
}

func (p *winProcess) Suspend() error { return p.each(ntSuspendProcess) }
func (p *winProcess) Resume() error  { return p.each(ntResumeProcess) }

func (p *winProcess) Close() {
	// Stop follow() first: it uses the job, the port and the tracked handle.
	p.closeOnce.Do(func() { close(p.stop) })
	<-p.followDone
	p.mu.Lock()
	if p.tracked != 0 {
		windows.CloseHandle(p.tracked)
		p.tracked = 0
	}
	p.mu.Unlock()
	if p.mainH != 0 {
		windows.CloseHandle(p.mainH)
		p.mainH = 0
	}
	p.closeJob()
}

func imageName(h windows.Handle) string {
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return ""
	}
	return filepath.Base(windows.UTF16ToString(buf[:n]))
}

func creationTime(h windows.Handle) time.Time {
	var c, e, k, u windows.Filetime
	if err := windows.GetProcessTimes(h, &c, &e, &k, &u); err != nil {
		return time.Time{}
	}
	return time.Unix(0, c.Nanoseconds())
}

// FindByName returns the PIDs of every running process with that image name.
func FindByName(name string) ([]int, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	var pids []int
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if strings.EqualFold(windows.UTF16ToString(e.ExeFile[:]), name) {
			pids = append(pids, int(e.ProcessID))
		}
	}
	// Any other error than the end of the list means "could not tell": an
	// empty answer would read as "not running".
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, fmt.Errorf("liste des process : %w", err)
	}
	return pids, nil
}

// KillByName terminates every process with that image name and waits up to
// timeout for them to be gone. It returns how many were killed.
func KillByName(name string, timeout time.Duration) (int, error) {
	pids, err := FindByName(name)
	if err != nil {
		return 0, err
	}
	var errs []error
	killed := 0
	for _, pid := range pids {
		h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
		if err != nil {
			errs = append(errs, fmt.Errorf("pid %d : %w", pid, err))
			continue
		}
		// The PID may have been reused since the listing: check who it is now.
		if !strings.EqualFold(imageName(h), name) {
			windows.CloseHandle(h)
			continue
		}
		if err := windows.TerminateProcess(h, 1); err != nil {
			errs = append(errs, fmt.Errorf("pid %d : %w", pid, err))
		} else if ev, _ := windows.WaitForSingleObject(h, uint32(timeout.Milliseconds())); ev != windows.WAIT_OBJECT_0 {
			errs = append(errs, fmt.Errorf("pid %d toujours présent après %s", pid, timeout))
		} else {
			killed++
		}
		windows.CloseHandle(h)
	}
	return killed, errors.Join(errs...)
}
