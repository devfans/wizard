package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"strconv"
	"strings"
	"syscall"

	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"golang.org/x/term"

	"github.com/devfans/envconf"
	"github.com/urfave/cli/v2"
)

var (
	manager    *Manager
	_KILL_WAIT = 0
)

// Attribute defines a single SGR Code
type Attribute int

// Foreground text colors
const (
	FgBlack Attribute = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

func Fatal(msg string, args ...interface{}) {
	fmt.Printf("\x1b[%dm> %s\x1b[0m\n", FgRed, fmt.Sprintf(msg, args...))
	os.Exit(2)
}

func Info(msg string, args ...interface{}) {
	fmt.Printf("\x1b[%dm> %s\x1b[0m\n", FgCyan, fmt.Sprintf(msg, args...))
}

type Manager struct {
	pid     int
	logging bool
	process *os.Process

	config *envconf.Config
}

func (m *Manager) Init() {
	var err error
	// Validate work directory
	dir := m.config.Get("dir")
	if dir == "" {
		dir = "."
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		Fatal("Invalid work directory: %v", err)
	}
	m.config.Put("dir", dir)

	// Validate command path
	cmd := m.config.Get("cmd")
	if cmd == "" {
		Fatal("Process command line is not specified")
	}

	// Validate pid file path
	pid := m.config.Get("pid")
	if pid == "" {
		pid = ".pid"
	}
	m.config.Put("pid", path.Join(dir, pid))

	// Validate log file path
	logging := m.config.Get("logging")
	if logging != "" && strings.ToLower(logging) == "false" {
		m.logging = false
	}
	m.config.Put("logging", m.logging)

	if m.logging {
		logFile := m.config.Get("log")
		if logFile == "" {
			logFile = "app.log"
		}
		m.config.Put("log", path.Join(dir, logFile))
	}
}

func (m *Manager) getEnv() []string {
	envs := make([]string, 0)
	for key, value := range m.config.GetSection("env") {
		envs = append(envs, fmt.Sprintf("%v=%v", key, value))
	}
	return envs
}

func (m *Manager) findProcess() bool {
	pidFile := m.config.Get("pid")
	pidData, err := ioutil.ReadFile(pidFile)
	if err != nil {
		Info("Failed to read pid file: %v", err)
		return false
	}

	m.pid, err = strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		Info("Failed to read process pid: %v", err)
		return false
	}

	m.process, err = os.FindProcess(m.pid)
	if err != nil {
		Info("Failed to find process with pid: %v", err)
		return false
	}

	err = m.process.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}
	return true
}

func (m *Manager) parseCommand(command string) (string, []string) {
	args := make([]string, 0)

	quoting := false
	var quote, cache string

	for _, r := range command {
		c := string(r)
		if quoting {
			if c == quote {
				// end quoting
				if len(cache) > 0 {
					args = append(args, cache)
					cache = ""
				}
				quoting = false
			} else {
				cache += c
			}
		} else {
			if c == "\"" || c == "'" {
				// open quoting
				quoting = true
				quote = c
				// append last cache
				if len(cache) > 0 {
					args = append(args, cache)
					cache = ""
				}
			} else if c == " " {
				// finish the cache
				if len(cache) > 0 {
					args = append(args, cache)
					cache = ""
				}
			} else {
				cache += c
			}
		}
	}

	if len(cache) > 0 {
		args = append(args, cache)
	}
	if len(args) > 0 {
		return args[0], args[1:]
	}
	return "", args
}

func (m *Manager) spawn(input bool) {
	pidFile := m.config.Get("pid")
	pidFileObject, err := os.OpenFile(pidFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		Fatal("Failed to create pid file %v: %v", pidFile, err)
	}
	defer pidFileObject.Close()

	command := m.config.Get("cmd")
	exe, args := m.parseCommand(command)
	exe = strings.Replace(exe, "~", os.Getenv("HOME"), 1)
	_, err = exec.LookPath(exe)
	if err != nil {
		exe, err = filepath.Abs(exe)
		if err != nil {
			Fatal("Failed to find executable %v: %v", exe, err)
		}
	}
	Info("Wizard is launching process with below command and args")
	argstr, _ := json.Marshal(args)
	Info("%v %s", exe, string(argstr))

	cmd := exec.Command(exe, args...)
	// Append extra env vars
	envars := m.getEnv()
	if len(envars) > 0 {
		Info("Extra env vars: %v", envars)
		cmd.Env = append(os.Environ(), envars...)
	}

	// Add logging
	if m.logging {
		logFile := m.config.Get("log")
		if _, err := os.Stat(logFile); err == nil {
			err = os.Rename(logFile, logFile+string(time.Now().Format(time.RFC3339)))
			if err != nil {
				Info("Failed to rename old log file %v: %v", logFile, err)
			}
		}

		logFileObject, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE, 0664)
		if err != nil {
			Fatal("Failed to create log file %v: %v", logFile, err)
		}
		defer logFileObject.Close()

		cmd.Stdout = logFileObject
		cmd.Stderr = logFileObject
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Pdeathsig: syscall.SIGKILL,
		Setpgid: true,
	}

	if input {
		data, err := ReadInput("input")
		if err != nil {
			Fatal("Failed to read input: %v", err)
		}
		cmd.Stdin = bytes.NewReader(data)
	}

	err = cmd.Start()
	if err != nil {
		Fatal("Failed to spawn the process: %v", err)
	}

	m.pid = cmd.Process.Pid
	count, err := pidFileObject.WriteString(strconv.Itoa(m.pid))
	if err != nil {
		Info("Failed to save pid file: %v", err)
	}
	if err = pidFileObject.Truncate(int64(count)); err != nil {
		Info("Failed to truncate pid file: %v", err)
	}
	pidFileObject.Sync()
}

func (m *Manager) Start(input bool) {
	if m.findProcess() {
		Info("Process is already running")
		return
	}
	m.spawn(input)
	Info("Process is started")
}

func (m *Manager) Stop() {
	if m.findProcess() {
		err := m.process.Signal(syscall.SIGTERM)
		if err != nil {
			Fatal("Error encountered: %v", err)
		}
		count := 0
		for {
			err = m.process.Signal(syscall.Signal(0))
			if err != nil {
				break
			}
			if count > _KILL_WAIT && _KILL_WAIT > 0 {
				Info("Force process to exit now...")
				err = m.process.Kill()
				if err != nil {
					Info("Error encountered: %v", err)
				}
				break
			} else {
				time.Sleep(10 * time.Millisecond)
				count++
			}
		}
	}
	Info("Process is stopped")
}

func (m *Manager) Status() {
	if m.findProcess() {
		Info("Process is running.")
	} else {
		Info("Process is stopped.")
	}
}

func ReadInput(name string) ([]byte, error) {
	fmt.Printf("Enter %s: \n", name)
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	return passphrase, nil
}

func initialize(ctx *cli.Context) (err error) {
	config := envconf.NewConfig(ctx.String("config"))
	manager = &Manager{config: config, logging: true}
	manager.Init()
	return
}

func start(ctx *cli.Context) (err error) {
	manager.Start(ctx.Bool("i"))
	return
}

func status(ctx *cli.Context) (err error) {
	manager.Status()
	return
}

func stop(ctx *cli.Context) (err error) {
	_KILL_WAIT = ctx.Int("f") * 100
	manager.Stop()
	return
}

func restart(ctx *cli.Context) (err error) {
	stop(ctx)
	time.Sleep(time.Duration(ctx.Int("w")) * time.Second)
	start(ctx)
	return
}

func main() {
	app := &cli.App{
		Name:  "wizard",
		Usage: "The awesome process manager",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "c",
				Aliases: []string{"config", "conf"},
				Value:   ".wiz",
				Usage:   "wizard configuration file",
			},
		},
		Before: initialize,
		Commands: []*cli.Command{
			&cli.Command{
				Name:   "start",
				Usage:  "Launch the process",
				Action: start,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "i",
						Aliases: []string{"input", "stdin"},
						Usage:   "input from stdin",
					},
				},
			},
			&cli.Command{
				Name:   "stop",
				Usage:  "Stop the running process",
				Action: stop,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "f",
						Aliases: []string{"force"},
						Usage:   "seconds later to force the process to exit if process wont stop and specified bigger than 0",
					},
				},
			},
			&cli.Command{
				Name:   "status",
				Usage:  "Check status of the process",
				Action: status,
			},
			&cli.Command{
				Name:   "restart",
				Usage:  "Restart the process",
				Action: restart,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "f",
						Aliases: []string{"force"},
						Usage:   "seconds later to force the process to exit if process wont stop and specified bigger than 0",
					},
					&cli.IntFlag{
						Name:    "w",
						Aliases: []string{"wait"},
						Value:   1,
						Usage:   "seconds to wait before start the process again after it's stopped",
					},
					&cli.BoolFlag{
						Name:    "i",
						Aliases: []string{"input", "stdin"},
						Usage:   "input from stdin",
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		Fatal("Failed: %v", err)
	}
}
