# wizard
A light-weight process bootstrap to start/stop/check a process 

## Get Started

```
// wizard start/stop/status

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
```

## Configuration

Default config file `.wiz` to specify a config file `wizard start/stop/status -c app.wiz`

Sample configuration
```
[main]
dir = .
log = app.log
pid = app.pid
command = server-run -c 1 -s 2

```





