# wizard
A light-weight process bootstrap to start/stop/check a process 

## Get Started

```
// wizard start/stop/restart/status

IceWall:bin stefan$ ./wizard start
2019/08/23 19:32:56 Process is started

IceWall:bin stefan$ ./wizard status
2019/08/23 19:32:59 Process is running.

IceWall:bin stefan$ ./wizard start
2019/08/23 19:33:04 Process is already running

IceWall:bin stefan$ ./wizard stop
2019/08/23 19:33:07 Process is stopped

IceWall:bin stefan$ ./wizard status
2019/08/23 19:33:09 Process is stopped.

IceWall:bin stefan$ ./wizard restart -w 2000 // wait 2000 millisecs after stopped before a restart

```

## Configuration

Default config file `.wiz` to specify a config file `wizard start/stop/restart/status -c app.wiz`

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




