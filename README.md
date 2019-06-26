# tmux-compose
Orchestrate the creation of tmux sessions with dependencies between commands in windows and panes.

This project is like a mash-up of docker-compose and teamocil/tmuxinator for tmux, hence the name "tmux-compose".

You create YAML config files that detail what windows and panes should be created and any commands that should be run in them. Then, you can setup dependencies between panes and windows to orchestrate the order in which the commands are run.

### example.yml
```yaml
name: example
dir: ~/project
windows:

  # Start a database
  - name: database
    panes:
      - cmd: service postgresql start
        readycheck:
          test: pg_isready -h localhost -p 5432 -U postgres
          interval: 3s
          retries: 3

  # Run a program that must start after the database is ready
  - panes:
      - cmd: ./myprogram
        depends_on: ["database"]
```

Bring up a tmux session:
```bash
tmux-compose -f example.yml up
```

Teardown a tmux session:
```bash
tmux-compose -f example.yml down
```

### Installation

tmux-compose was built with Go.

```bash
go get github.com/kevinms/tmux-compose.git
```

#### Build from source

```bash
git clone https://github.com/kevinms/tmux-compose.git
cd tmux-compose
go install
```

Go code can easily compile for other OSes, but I have only tested running it in Linux.

### Project
Example showing all options for the root node of the config file
```yaml
name: example
dir: /path/to/project
pre_cmd: touch example.tmp
post_cmd: rm example.tmp
windows:
  - name: code
    panes:
    - cmd: vim
  - panes:
    - cmd: top
```

### Windows
Example showing all options being used for a window:
```yaml
name: example
windows:
  - name: My Window
    dir: ~/project
    focus: true
    layout: main-vertical
    depends_on: ["thing1", "thing2"]
    panes:
      - cmd: vim
      - cmd: sleep 5
```

### Panes
Example showing all options being used for a pane:
```yaml
name: example
windows:
  - panes:
    - name: My Pane
      dir: ~/project
      cmd: python -m SimpleHTTPServer 8000
      focus: true
      readycheck:
        test: ping -c1 domain.net
        interval: 3s
        retries: 10
      depends_on: ["thing1", "thing2"]
```

#### Directly Inspired By:

* docker-compose
* teamocil
* tmuxinator
* tmuxstart
