package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/shortcut"
	"github.com/davidp57/process-monitor/internal/tui"
	"github.com/davidp57/process-monitor/internal/wizard"
)

// menu is what a double-click on pm.exe shows: the commands, chosen from a
// list, for whoever does not type command lines. The wizard comes first.
func menu() int {
	for {
		fmt.Println("\n" + i18n.T("menu.title", version))
		values, answer, err := tui.Ask(&wizard.Question{Kind: wizard.Choice, Prompt: i18n.T("menu.q"), Options: []wizard.Option{
			{Value: "new", Label: i18n.T("menu.new")},
			{Value: "shortcut", Label: i18n.T("menu.shortcut")},
			{Value: "check", Label: i18n.T("menu.check")},
			{Value: "run", Label: i18n.T("menu.run")},
			{Value: "watch", Label: i18n.T("menu.watch")},
			{Value: "quit", Label: i18n.T("menu.quit")},
		}}, "")
		if err != nil || answer != tui.Answered || values[0] == "quit" {
			return 0
		}
		cmd := values[0]
		if cmd == "new" {
			run([]string{"new"})
			continue
		}
		path, ok := askConfig()
		if !ok {
			continue
		}
		if cmd == "shortcut" && !shortcut.HasSection(path) && !addSection(path) {
			continue
		}
		if cmd == "shortcut" && !shortcut.HasUI(path) && !addUI(path) {
			continue
		}
		if cmd == "shortcut" && !addIcons(path) {
			continue
		}
		profile := ""
		if cmd != "shortcut" { // pm shortcut makes every profile's shortcut
			if profile, ok = askProfile(path); !ok {
				continue
			}
		}
		rc := run(menuArgs(cmd, path, profile))
		if cmd == "run" || cmd == "watch" {
			return rc // the session has ended, and with it what the window was for
		}
	}
}

// menuArgs is the command line the menu runs.
func menuArgs(cmd, path, profile string) []string {
	argv := []string{cmd, path}
	if profile != "" {
		argv = append(argv, "--profile", profile)
	}
	return argv
}

// askConfig asks for a config file, suggesting one next to pm.exe or in the
// folder above (where the configs usually are).
func askConfig() (string, bool) {
	q := &wizard.Question{Kind: wizard.Path, Prompt: i18n.T("menu.q_config")}
	if found := nearbyConfig(filepath.Dir(self())); found != "" {
		q.Default = []string{found}
	}
	errMsg := ""
	for {
		values, answer, err := tui.Ask(q, errMsg)
		if err != nil || answer != tui.Answered {
			return "", false
		}
		path := strings.Trim(strings.TrimSpace(values[0]), `"`)
		if path == "" && len(q.Default) > 0 {
			path = q.Default[0]
		}
		if _, err := os.Stat(path); err != nil {
			errMsg = i18n.T("wizard.not_found", path)
			continue
		}
		return path, true
	}
}

// nearbyConfig is the first .yaml file in dir or its parent, or "".
func nearbyConfig(dir string) string {
	for _, d := range []string{dir, filepath.Dir(dir)} {
		matches, _ := filepath.Glob(filepath.Join(d, "*.yaml"))
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// addSection offers to add the shortcuts section a config lacks: from the
// menu, nobody is expected to open the YAML. It reports whether it was added.
func addSection(path string) bool {
	ask := func(q *wizard.Question) (string, bool) {
		values, answer, err := tui.Ask(q, "")
		if err != nil || answer != tui.Answered || len(values) == 0 {
			return "", false
		}
		return values[0], true
	}
	yn := []wizard.Option{{Value: "yes", Label: i18n.T("wizard.yes")}, {Value: "no", Label: i18n.T("wizard.no")}}
	if v, ok := ask(&wizard.Question{Kind: wizard.YesNo, Prompt: i18n.T("menu.q_add_section"), Help: i18n.T("menu.h_add_section"),
		Options: yn}); !ok || v != "yes" {
		return false
	}
	mode, ok := ask(&wizard.Question{Kind: wizard.Choice, Prompt: i18n.T("menu.q_mode"), Options: []wizard.Option{
		{Value: config.ModeRun, Label: i18n.T("menu.mode_run")},
		{Value: config.ModeWatch, Label: i18n.T("menu.mode_watch")},
	}})
	if !ok {
		return false
	}
	q := wizard.ShortcutFolders()
	chosen, answer, err := tui.Ask(&q, "")
	if err != nil || answer != tui.Answered || len(chosen) == 0 {
		return false
	}
	var folders []string
	for _, f := range chosen {
		if f == "other" {
			if f, ok = ask(&wizard.Question{Kind: wizard.Path, Prompt: i18n.T("wizard.q.shortcuts_folder")}); !ok || f == "" {
				return false
			}
		}
		folders = append(folders, f)
	}
	if err := shortcut.AddSection(path, mode, folders); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.config_error", err))
		return false
	}
	fmt.Println(i18n.T("menu.section_added", path))
	return true
}

// addUI asks how sessions of a config that does not say show themselves,
// and writes the answer: the shortcut opens minimized for the icon.
func addUI(path string) bool {
	values, answer, err := tui.Ask(&wizard.Question{Kind: wizard.Choice, Prompt: i18n.T("menu.q_ui"), Options: []wizard.Option{
		{Value: config.UITray, Label: i18n.T("menu.ui_tray")},
		{Value: config.UIConsole, Label: i18n.T("menu.ui_console")},
	}}, "")
	if err != nil || answer != tui.Answered {
		return false
	}
	if err := shortcut.SetUI(path, values[0]); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.config_error", err))
		return false
	}
	fmt.Println(i18n.T("menu.ui_added", values[0], path))
	return true
}

// addIcons offers an icon to each profile that has none, among the .ico
// files next to the config, before its shortcuts are made. It reports
// false when the player gave up.
func addIcons(path string) bool {
	profiles, err := config.Profiles(path)
	if err != nil {
		return true // pm shortcut says what is wrong with the file
	}
	if len(profiles) == 0 {
		profiles = []string{""}
	}
	dir := filepath.Dir(path)
	found := wizard.FindIcons(dir)
	for _, p := range profiles {
		c, err := config.Load(path, p)
		if err != nil || c.Icon != "" {
			continue
		}
		label := p
		if label == "" {
			label = c.Name
		}
		q := wizard.IconQuestion(label, c.Name, dir, found)
		values, answer, err := tui.Ask(&q, "")
		if err != nil || answer != tui.Answered {
			return false
		}
		icon := values[0]
		if icon == wizard.IconNone {
			continue
		}
		if icon == wizard.IconOther {
			typed := wizard.IconPath(label, func(f string) bool { _, err := os.Stat(f); return err == nil })
			errMsg := ""
			for {
				values, answer, err := tui.Ask(&typed, errMsg)
				if err != nil || answer != tui.Answered {
					return false
				}
				if err := typed.Check(values); err != nil {
					errMsg = err.Error()
					continue
				}
				icon = values[0]
				break
			}
		}
		if err := shortcut.SetIcon(path, p, icon); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Println(i18n.T("menu.icon_added", label, icon))
	}
	return true
}

// askProfile asks which profile, when the config has some.
func askProfile(path string) (string, bool) {
	names, err := config.Profiles(path)
	if err != nil || len(names) == 0 {
		return "", true // no profile to choose; pm says what is wrong with the file
	}
	opts := make([]wizard.Option, len(names))
	for i, n := range names {
		opts[i] = wizard.Option{Value: n, Label: n}
	}
	values, answer, err := tui.Ask(&wizard.Question{Kind: wizard.Choice, Prompt: i18n.T("menu.q_profile"), Options: opts}, "")
	if err != nil || answer != tui.Answered {
		return "", false
	}
	return values[0], true
}
