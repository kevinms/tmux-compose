# tmux-compose
Orchestrate the creation of tmux sessions with dependencies between commands in windows and panes.

This project is like a mash-up of docker-compose and teamocil/tmuxinator for tmux, hence the name "tmux-compose".

You create YAML config files that detail what windows and panes should be created and any commands that should be run in them. Then, you can setup dependencies between panes and windows to orchestrate the order in which the commands are run.

- [Example Usage](#exampleyml)
- [Installation](#installation)
  - [Build from source](#build-from-source)
- [Project Options](#project)
- [Windows Options](#windows)
- [Panes Options](#panes)


## example.yml
```yaml
dir: ~/project
sessions:
  - name: example
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

## Installation

##### Prebuilt binaries for stable releases
Prebuilt binaries for multiple platforms can be downloaded from the [releases page](https://github.com/kevinms/tmux-compose/releases).

##### Automated build from source
tmux-compose was built with Go. If you already have Go setup, you `go get` the utility:

```bash
go get github.com/kevinms/tmux-compose
```

##### Manually build from source

```bash
git clone https://github.com/kevinms/tmux-compose.git
cd tmux-compose
go install
```

Go code can easily compile for other OSes, but this has only been tested on Linux.

## Project
Example showing all options for the root node of the config file
```yaml
dir: /path/to/project
up_pre_cmd: (date; echo start) > run.log
up_post_cmd: (date; echo done) >> run.log
down_pre_cmd: touch example.tmp
down_post_cmd: rm example.tmp
sessions:
  - name: example
    windows:
      - name: code
        panes:
        - cmd: vim
      - panes:
        - cmd: top
```

## Sessions
Example showing all options being used for a window:
```yaml
sessions:
  - name: example
    dir: ~/project
    readycheck:
      test: ping -c1 domain.net
      interval: 3s
      retries: 10
    depends_on: ["thing1", "thing2"]
    windows:
      - name: code
        panes:
        - cmd: vim
      - panes:
        - cmd: top
```

## Windows
Example showing all options being used for a window:
```yaml
sessions:
  - name: example
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

## Panes
Example showing all options being used for a pane:
```yaml
sessions:
  - name: example
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

## Directly Inspired By:

* docker-compose
* teamocil
* tmuxinator
* tmuxstart
