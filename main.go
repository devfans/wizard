package main

import (
  "os"
  "os/exec"
  "log"
  "flag"
  "path"
  "time"
  "fmt"

  "syscall"
  "strings"
  "strconv"

  "io/ioutil"
  "path/filepath"
  "encoding/json"

  "github.com/devfans/envconf"
)

type Manager struct {
  pid           int
  logging       bool
  process       *os.Process

  config        *envconf.Config
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
    log.Fatalln("Invalid work directory:", err)
  }
  m.config.Put("dir", dir)

  // Validate command path
  cmd := m.config.Get("cmd")
  if cmd == "" {
    log.Fatalln("Command not specified")
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

func (m* Manager) getEnv() []string {
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
    log.Printf("Failed to read pid file: %v", pidFile)
    return false
  }

  m.pid, err = strconv.Atoi(string(pidData))
  if err != nil {
    log.Println("Failed to read process pid")
    return false
  }

  m.process, err = os.FindProcess(m.pid)
  if err != nil {
    log.Printf("Failed to find process with pid:", m.pid)
    return false
  }

  err = m.process.Signal(syscall.Signal(0))
  if err != nil {
    return false
  }
  return true
}

func(m *Manager) parseCommand(command string) (string, []string) {
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

func (m *Manager) spawn () {
  pidFile := m.config.Get("pid")
  pidFileObject, err := os.OpenFile(pidFile, os.O_RDWR|os.O_CREATE, 0755)
  if err != nil {
    log.Fatalf("Failed to create pid file %v Error %v", pidFile, err)
  }
  defer pidFileObject.Close()

  command := m.config.Get("cmd")
  exe, args := m.parseCommand(command)
  exe = strings.Replace(exe, "~", os.Getenv("HOME"), 1)
  _, err = exec.LookPath(exe)
  if err != nil {
    exe, err = filepath.Abs(exe)
    if err != nil {
      log.Fatalf("Failed to find executable: %v", exe)
    }
  }
  log.Println("Wizard is launching process with below command and args")
  argstr, _ := json.Marshal(args)
  log.Printf("%v %s", exe, argstr)

  cmd := exec.Command(exe, args...)
  // Append extra env vars
  envars := m.getEnv()
  if len(envars) > 0 {
    log.Println("Extra env vars:", envars)
    cmd.Env = append(os.Environ(), envars...)
  }

  // Add logging
  if m.logging {
    logFile := m.config.Get("log")
    if _, err := os.Stat(logFile); err == nil {
      err = os.Rename(logFile, logFile + string(time.Now().Format(time.RFC3339)))
      if err != nil {
        log.Println("Failed to rename old log file:", logFile)
      }
    }

    logFileObject, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE, 0755)
    if err != nil {
      log.Fatalln("Failed to create log file:", logFile)
    }
    defer logFileObject.Close()

    cmd.Stdout = logFileObject
    cmd.Stderr = logFileObject
  }
  err = cmd.Start()
  if err != nil {
    log.Fatalln("Failed to spawn the process, error:", err)
  }
  m.pid = cmd.Process.Pid

  _, err = pidFileObject.WriteString(strconv.Itoa(m.pid))
  if err != nil {
    log.Println("Failed to save pid file:", err)
  }
  pidFileObject.Sync()
}

func (m *Manager) Start() {
  if m.findProcess() {
    log.Println("Process is already running")
    return
  }
  m.spawn()
  log.Println("Process is started")
}

func (m *Manager) Stop() {
  if m.findProcess() {
    err := m.process.Kill()
    if err != nil {
      log.Println("Error encountered", err)
      return
    }
  }
  log.Println("Process is stopped")
}

func (m *Manager) Status() {
  if m.findProcess() {
    log.Println("Process is running.")
  } else {
    log.Println("Process is stopped.")
  }
}

func main() {
  if len(os.Args) < 2 {
    log.Fatalln("Subcommand is required: start/stop/restart/status")
  }
  subcommand := os.Args[1]

  flagSet := flag.NewFlagSet("subcommand", flag.ExitOnError)
  configFile := flagSet.String("c", ".wiz", "wizard config file")
  flagSet.Parse(os.Args[2:])

  config := envconf.NewConfig(*configFile)
  manager := Manager { config: config, logging: true }
  manager.Init()

  switch strings.ToLower(subcommand) {
  case "start":
    manager.Start()
  case "stop":
    manager.Stop()
  case "status":
    manager.Status()
  case "restart":
    manager.Stop()
    time.Sleep(500 * time.Millisecond)
    manager.Start()
  default:
    log.Fatalln("Subcommand should be start/stop/restart/status")
  }
}
