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

func coalesce(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

func main() {
	var override string
	flag.StringVar(&override, "s", "", "Override the config files session name.")
	flag.Parse()

	if flag.Arg(0) == "" {
		fmt.Println("Usage: ", os.Args[0], "[-s <name>] <config file>")
		os.Exit(1)
	}
	flag.Arg(0)

	data, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	var project Project

	err = yaml.UnmarshalStrict(data, &project)
	if err != nil {
		log.Fatal(err)
	}

	session := coalesce(override, project.Name, "new")

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
		target := fmt.Sprintf("%s:%d", session, wi)
		if w.Focus {
			SelectWindow(target)
		}
	}
}
