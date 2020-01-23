package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Pane struct {
	Object  `yaml:",inline"`
	Dir     string
	Focus   bool
	Cmd     string
	KillCmd string `yaml:"kill_cmd"`
	target  string
}

type Window struct {
	Object `yaml:",inline"`
	Dir    string
	Focus  bool
	Layout string
	Panes  []*Pane
}

type Session struct {
	Object  `yaml:",inline"`
	Dir     string
	Windows []*Window
	started bool
}

type Project struct {
	Dir         string
	UpPreCmd    string `yaml:"up_pre_cmd"`
	UpPostCmd   string `yaml:"up_post_cmd"`
	DownPreCmd  string `yaml:"down_pre_cmd"`
	DownPostCmd string `yaml:"down_post_cmd"`
	Sessions    []*Session
}

var gShellArgs []string
var gRestart bool

func run(format string, args ...interface{}) error {
	cmdStr := fmt.Sprintf(format, args...)
	cmd := exec.Command(gShellArgs[0], append(gShellArgs[1:], cmdStr)...)

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

func shellInDir(dir, cmd string) {
	shell("cd %s;%s", coalesce(dir, "."), cmd)
}

func NewWindow(session *Session, dir string) {
	if session.started {
		shell("tmux new-window -d -t %s -c %s", session.Name, dir)
	} else {
		shell("tmux new-session -d -s %s -c %s", session.Name, dir)
		session.started = true
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
	run("tmux kill-session -t %s", session)
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

func (project *Project) getDir(s *Session, w *Window, paneIndex int) string {
	if paneIndex > len(w.Panes) {
		log.Fatal("Pane index out of bounds?!")
	}

	if paneIndex == 0 && len(w.Panes) == 0 {
		// The window has no explicit panes
		return coalesce(w.Dir, s.Dir, project.Dir, ".")
	}

	return coalesce(w.Panes[paneIndex].Dir, w.Dir, s.Dir, project.Dir, ".")
}

func (p *Pane) Run() {
	if gRestart && p.KillCmd == "" {
		return
	}
	SendLine(p.target, p.Cmd)
}

func (w *Window) Run() {
}

func (s *Session) Run() {
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

func (s *Session) DoReadyCheck() {
	for {
		ready := true

		for _, w := range s.Windows {
			if w == nil {
				continue
			}
			if !w.IsReady() {
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

func (project *Project) up() {
	if project.UpPreCmd != "" {
		shellInDir(project.Dir, project.UpPreCmd)
	}

	// Spawn all the sessions/windows/panes
	for _, s := range project.Sessions {
		for wi, w := range s.Windows {
			if w == nil {
				continue
			}
			target := fmt.Sprintf("%s:%d", s.Name, wi)
			dir := project.getDir(s, w, 0)

			NewWindow(s, dir)

			for pi, p := range w.Panes {
				if p == nil {
					continue
				}
				p.target = fmt.Sprintf("%s:%d.%d", s.Name, wi, pi)
				dir := project.getDir(s, w, pi)
				if pi > 0 {
					NewPane(target, dir)
				}
			}

			SelectLayout(target, w.Layout)
		}
	}

	// Set which window has focus
	for _, s := range project.Sessions {
		for wi, w := range s.Windows {
			if w == nil {
				continue
			}
			if w.Focus {
				target := fmt.Sprintf("%s:%d", s.Name, wi)
				SelectWindow(target)
			}
		}
	}

	// Run the commands concurrently
	for _, s := range project.Sessions {
		if s == nil {
			continue
		}
		addRunner(s)
		for _, w := range s.Windows {
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
	}
	runAll()

	if project.UpPostCmd != "" {
		shellInDir(project.Dir, project.UpPostCmd)
	}
}

func (project *Project) down() {
	if project.DownPreCmd != "" {
		shellInDir(project.Dir, project.DownPreCmd)
	}

	for _, s := range project.Sessions {
		KillSession(s.Name)
	}

	if project.DownPostCmd != "" {
		shellInDir(project.Dir, project.DownPostCmd)
	}
}

func (project *Project) restart() {
	// Run the commands concurrently
	for _, s := range project.Sessions {
		for wi, w := range s.Windows {
			if w == nil {
				continue
			}
			for pi, p := range w.Panes {
				if p == nil {
					continue
				}
				p.target = fmt.Sprintf("%s:%d.%d", s.Name, wi, pi)

				if p.KillCmd != "" {
					SendLine(p.target, p.KillCmd)
				}
			}
		}
	}

	// Used in each Panel's Run() method to know if we are performing a restart.
	// Only commands with a KillCmd will be restarted.
	gRestart = true

	// Run the commands concurrently
	for _, s := range project.Sessions {
		if s == nil {
			continue
		}
		addRunner(s)
		for _, w := range s.Windows {
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
	}
	runAll()
}

func getDefaultShell() string {
	sh := os.Getenv("SHELL")

	if sh == "" {
		return "/bin/sh -c"
	}

	return sh + " -c"
}

func main() {
	var composeFile string
	var shellArgs string
	flag.StringVar(&composeFile, "f", "tmux-compose.yml", "Specify an alternate compose file")
	flag.StringVar(&shellArgs, "shell", getDefaultShell(), "Specify an alternate shell path")
	flag.Parse()

	fmt.Printf("Using shell args: %s\n", shellArgs)
	gShellArgs = strings.Split(shellArgs, " ")

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

	action := flag.Arg(0)

	switch action {
	case "up":
		project.up()
	case "down":
		project.down()
	case "restart":
		project.restart()
	}
}
