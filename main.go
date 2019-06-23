package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type Project struct {
	Name    string
	Dir     string
	Windows []struct {
		Name   string
		Dir    string
		Layout string
		Focus  bool
		Panes  []string
	}
}

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)

	fmt.Println(name, strings.Join(args[:], " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Output: %s", out)
		log.Fatal(err)
	}
}

var sessionStarted bool

func NewWindow(session, dir string) {
	if sessionStarted {
		run("tmux", "new-window", "-d", "-t "+session, "-c "+dir)
	} else {
		run("tmux", "new-session", "-d", "-s "+session, "-c "+dir)
		sessionStarted = true
	}
}

func NewPane(target, dir string) {
	run("tmux", "split-window", "-t "+target, "-c "+dir)
}

func SelectWindow(target string) {
	run("tmux", "select-window", "-t "+target)
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
	run("tmux", "select-layout", "-t "+target, layout)
}

func SendLine(target, text string) {
	if text == "" {
		return
	}
	run("tmux", "send-keys", "-l", "-t "+target, text)
	run("tmux", "send-keys", "-R", "-t "+target, "Enter")
}

func KillSession(session string) {
	run("tmux", "kill-session", "-t "+session)
}

func coalesce(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

func up(session string, project *Project) {
	for wi, w := range project.Windows {
		target := fmt.Sprintf("%s:%d", session, wi)
		dir := coalesce(w.Dir, project.Dir, ".")

		NewWindow(session, dir)

		for pi, p := range w.Panes {
			if pi > 0 {
				NewPane(target, dir)
			}
			SendLine(target, p)
		}

		SelectLayout(target, w.Layout)
	}

	for wi, w := range project.Windows {
		if w.Focus {
			target := fmt.Sprintf("%s:%d", session, wi)
			SelectWindow(target)
		}
	}
}

func down(session string, project *Project) {
	KillSession(session)
}

var defaultFile = "tmux-compose.yml"

func main() {
	var overrideName string
	var composeFile string
	flag.StringVar(&overrideName, "s", "", "Override the config files session name.")
	flag.StringVar(&composeFile, "f", defaultFile, "Specify an alternate compose file")
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
