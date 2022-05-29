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
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h                              show help (default: false)
   -c value, --config value, --conf value  wizard configuration file (default: ".wiz")

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




