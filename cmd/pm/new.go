package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sys/windows"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/shortcut"
	"github.com/davidp57/process-monitor/internal/tui"
	"github.com/davidp57/process-monitor/internal/wizard"
)

// newConfig is pm new: the wizard, in the terminal.
func newConfig(a args) int {
	if !interactive() {
		fmt.Fprintln(os.Stderr, i18n.T("wizard.not_interactive"))
		return 2
	}
	docs, _ := windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
	env := wizard.Env{Home: os.Getenv("USERPROFILE"), Documents: docs, Lang: i18n.Lang(), Now: time.Now,
		Exists:   func(p string) bool { _, err := os.Stat(p); return err == nil },
		Profiles: config.Profiles,
		Icons:    wizard.FindIcons}
	w := wizard.New(env, a.path)
	log, err := logx.New("", "", 0, func(l logx.Line) { fmt.Println(l) })
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.log_error", err))
		return 1
	}
	r := &wizard.Runner{W: w, Log: log, PM: self()}
	fmt.Println(i18n.T("wizard.welcome"))

	errMsg := ""
	for {
		s := w.Next()
		switch {
		case s.Question != nil:
			values, answer, err := tui.Ask(s.Question, errMsg)
			errMsg = ""
			switch {
			case err != nil || answer == tui.Quit:
				return abandon(r)
			case answer == tui.Back:
				if !w.Back() && w.ActionsRan() {
					errMsg = i18n.T("wizard.no_back")
				}
			default:
				if err := w.Answer(values); err != nil {
					errMsg = err.Error()
				}
			}
		case s.Action != nil:
			if s.Action.Message != "" {
				fmt.Println("\n" + s.Action.Message)
			}
			// Ctrl+C is held while a step runs: stopping pm while the game is
			// open would leave the game running and the files half put back.
			// Quitting the game ends the step.
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			res, err := r.Do(context.WithoutCancel(ctx), *s.Action)
			stop()
			if err != nil {
				fmt.Fprintln(os.Stderr, i18n.T("wizard.step_failed", err))
				if s.Action.Kind == wizard.ActApply && env.Exists(w.Plan().ConfigPath) {
					return 1 // the config is written: trying again would only find it there
				}
				var running *wizard.Running
				errors.As(err, &running)
				values, answer, _ := tui.Ask(retry(running), "")
				if answer != tui.Answered {
					return abandon(r)
				}
				if len(values) > 0 && values[0] == "stop" {
					if err := r.StopGame(running); err != nil {
						fmt.Fprintln(os.Stderr, err)
					}
				}
				continue
			}
			w.Done(res)
			if links, ok := res.([]shortcut.Link); ok {
				rc := writeShortcuts(links)
				fmt.Println("\n" + i18n.T("wizard.created", w.Plan().ConfigPath))
				return rc
			}
		default:
			fmt.Println(i18n.T("wizard.nothing_written"))
			return 0
		}
	}
}

// retry asks what to do after a failed step; when the game was found
// running, stopping it is one of the answers.
func retry(running *wizard.Running) *wizard.Question {
	q := &wizard.Question{Kind: wizard.Choice, Prompt: i18n.T("wizard.q.retry"),
		Options: []wizard.Option{{Value: "retry", Label: i18n.T("wizard.retry")}}}
	if running != nil {
		q.Options = append([]wizard.Option{{Value: "stop", Label: i18n.T("wizard.stop_game", running.Name)}}, q.Options...)
	}
	return q
}

// abandon stops the wizard: nothing written, the original files back.
func abandon(r *wizard.Runner) int {
	if err := r.Abandon(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(i18n.T("wizard.nothing_written"))
	return 1
}
