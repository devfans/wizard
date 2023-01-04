# wizard
A light-weight process bootstrap to manage a process 

## Get Started

```
NAME:
   wizard - The awesome process manager

USAGE:
   wizard [global options] command [command options] [arguments...]

COMMANDS:
   start    Launch the process
   stop     Stop the running process
   status   Check status of the process
   restart  Restart the process
   daemon   Wizard daemon process to watch specified processes
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   -c value, --config value, --conf value  wizard configuration file (default: ".wiz")
   --help, -h                              show help (default: false)

```

## Configuration

Default config file `.wiz`, to specify a config file `wizard -c app.wiz start/stop/restart/status`

Sample configuration
```
[main]
dir = .
log = app.log
pid = app.pid
cmd = server-run -c 1 -s 2

[env]
var1=value1
var2=value2

```

## Daemon

Wizard can start a daemon process to watch specified processes in `~/.wiz`

Sampe `~/.wiz` 

```
[main]
cmd = wizard daemon
interval = 1000 // specify daemon watch interval(ms)

[daemon]
~/app1
~/app2/dir2
~/app3/.wiz
```

Sample `~/app3/.wiz`

```
cmd = server-run -s -c
no_daemon = false // set true to skip daemon watching
```





