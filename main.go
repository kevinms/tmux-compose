package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"gopkg.in/yaml.v2"
)

type Pane struct {
	Object `yaml:",inline"`
	Dir    string
	Focus  bool
	Cmd    string
	target string
}

type Window struct {
	Object `yaml:",inline"`
	Dir    string
	Focus  bool
	Layout string
	Panes  []*Pane
}

type Project struct {
	Name    string
	Dir     string
	PreCmd  string `yaml:"pre_cmd"`
	PostCmd string `yaml:"post_cmd"`
	Windows []*Window
}

func run(format string, args ...interface{}) error {
	cmdStr := fmt.Sprintf(format, args...)
	cmd := exec.Command("/bin/sh", "-c", cmdStr)

	fmt.Println(cmdStr)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}

func shell(format string, args ...interface{}) {
	err := run(format, args...)
	if err != nil {
		log.Fatal(err)
	}
}

var sessionStarted bool

func NewWindow(session, dir string) {
	if sessionStarted {
		shell("tmux new-window -d -t %s -c %s", session, dir)
	} else {
		shell("tmux new-session -d -s %s -c %s", session, dir)
		sessionStarted = true
	}
}

func NewPane(target, dir string) {
	shell("tmux split-window -t %s -c %s", target, dir)
}

func SelectWindow(target string) {
	shell("tmux select-window -t %s", target)
}

func SelectLayout(target, layout string) {
	if layout == "" {
		return
	}
	switch layout {
	case "even-horizontal":
	case "even-vertical":
	case "main-horizontal":
	case "main-vertical":
	case "titled":
	default:
		log.Fatal("Bad layout: " + layout)
	}
	shell("tmux select-layout -t %s %s", target, layout)
}

func SendLine(target, text string) {
	if text == "" {
		return
	}
	shell("tmux send-keys -t %s '%s'", target, text)
	shell("tmux send-keys -R -t %s 'Enter'", target)
}

func KillSession(session string) {
	shell("tmux kill-session -t %s", session)
}

func SetEnvironment(session, key, value string) {
	shell("tmux set-environment -t %s %s %s", session, key, value)
}

func coalesce(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

func (project *Project) getDir(w *Window, paneIndex int) string {
	if paneIndex > len(w.Panes) {
		log.Fatal("Pane index out of bounds?!")
	}

	if paneIndex == 0 && len(w.Panes) == 0 {
		// The window has no explicit panes
		return coalesce(w.Dir, project.Dir, ".")
	}

	return coalesce(w.Panes[paneIndex].Dir, w.Dir, project.Dir, ".")
}

func (p *Pane) Run() {
	SendLine(p.target, p.Cmd)
}

func (w *Window) Run() {
}

func (w *Window) DoReadyCheck() {
	for {
		ready := true

		for _, p := range w.Panes {
			if p == nil {
				continue
			}
			if !p.IsReady() {
				ready = false
				time.Sleep(100 * time.Millisecond)
				break
			}
		}

		if ready {
			return
		}
	}
}

func (p *Pane) DoReadyCheck() {
	if p.ReadyCheck.Test == "" {
		return
	}

	for {
		if err := run(p.ReadyCheck.Test); err == nil {
			break
		}

		if p.ReadyCheck.Retries <= 0 {
			log.Fatal("Object test failed?!")
		} else {
			p.ReadyCheck.Retries--
			time.Sleep(p.ReadyCheck.Interval)
		}
	}
}

func shellInDir(dir, cmd string) {
	shell("cd %s;%s", coalesce(dir, "."), cmd)
}

func up(session string, project *Project) {
	if project.PreCmd != "" {
		shellInDir(project.Dir, project.PreCmd)
	}

	// Spawn all the windows/panes
	for wi, w := range project.Windows {
		if w == nil {
			continue
		}
		target := fmt.Sprintf("%s:%d", session, wi)
		dir := project.getDir(w, 0)

		NewWindow(session, dir)

		for pi, p := range w.Panes {
			if p == nil {
				continue
			}
			p.target = fmt.Sprintf("%s:%d.%d", session, wi, pi)
			dir := project.getDir(w, pi)
			if pi > 0 {
				NewPane(target, dir)
			}
		}

		SelectLayout(target, w.Layout)
	}

	// Run the commands concurrently
	for _, w := range project.Windows {
		if w == nil {
			continue
		}
		addRunner(w)
		for _, p := range w.Panes {
			if p == nil {
				continue
			}
			addRunner(p)
		}
	}
	runAll()

	// Set which window has focus
	for wi, w := range project.Windows {
		if w == nil {
			continue
		}
		if w.Focus {
			target := fmt.Sprintf("%s:%d", session, wi)
			SelectWindow(target)
		}
	}

	if project.PostCmd != "" {
		shellInDir(project.Dir, project.PostCmd)
	}
}

func down(session string, project *Project) {
	KillSession(session)
}

func main() {
	var overrideName string
	var composeFile string
	flag.StringVar(&overrideName, "s", "", "Override the config files session name.")
	flag.StringVar(&composeFile, "f", "tmux-compose.yml", "Specify an alternate compose file")
	flag.Parse()

	if flag.Arg(0) == "" {
		fmt.Println("Usage: ", os.Args[0], "[OPTIONS] <up> | <down>")
		os.Exit(1)
	}

	data, err := ioutil.ReadFile(composeFile)
	if err != nil {
		log.Fatal(err)
	}

	var project Project

	err = yaml.UnmarshalStrict(data, &project)
	if err != nil {
		log.Fatal(err)
	}

	session := coalesce(overrideName, project.Name, "new")
	action := flag.Arg(0)

	switch action {
	case "up":
		up(session, &project)
	case "down":
		down(session, &project)
	}
}
