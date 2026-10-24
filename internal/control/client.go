package control

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/davidp57/process-monitor/internal/lock"
	"github.com/davidp57/process-monitor/internal/logx"
)

// Client talks to one session's control channel.
type Client struct {
	Info Info
	http *http.Client
}

// List finds the sessions running: every info file next to the locks whose
// session answers. A file left by a session that died without removing it
// is removed when its pm is gone.
func List() ([]*Client, error) {
	dir, err := lock.Dir()
	if err != nil {
		return nil, err
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var out []*Client
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var info Info
		if json.Unmarshal(data, &info) != nil || info.Port == 0 {
			continue
		}
		c := &Client{Info: info, http: &http.Client{Timeout: 2 * time.Second}}
		if _, err := c.State(); err != nil {
			if !alive(info.PID) {
				os.Remove(f)
			}
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (c *Client) url(path string) string {
	return "http://127.0.0.1:" + strconv.Itoa(c.Info.Port) + path
}

func (c *Client) do(method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, c.url(path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set(TokenHeader, c.Info.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrUnreachable
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, ErrUnreachable
	}
	return resp, nil
}

// State asks the session where it is.
func (c *Client) State() (Status, error) {
	var st Status
	resp, err := c.do("GET", "/state", nil)
	if err != nil {
		return st, err
	}
	defer resp.Body.Close()
	return st, json.NewDecoder(resp.Body).Decode(&st)
}

// Log returns the session's recent lines.
func (c *Client) Log() ([]logx.Line, error) {
	var lines []logx.Line
	resp, err := c.do("GET", "/log", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return lines, json.NewDecoder(resp.Body).Decode(&lines)
}

// Command sends one of Stop, Pause, Resume, Restart, Detach.
func (c *Client) Command(cmd string) error {
	body, _ := json.Marshal(map[string]string{"command": cmd})
	resp, err := c.do("POST", "/command", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Events calls fn with each new line until ctx is done or the session ends.
func (c *Client) Events(ctx context.Context, fn func(logx.Line)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.url("/events"), nil)
	if err != nil {
		return err
	}
	req.Header.Set(TokenHeader, c.Info.Token)
	resp, err := (&http.Client{}).Do(req) // no timeout: the stream lasts
	if err != nil {
		return ErrUnreachable
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		data, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var l logx.Line
		if json.Unmarshal([]byte(data), &l) == nil {
			fn(l)
		}
	}
	return ctx.Err()
}
