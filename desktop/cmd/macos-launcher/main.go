package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const expectedVersion = "3.8-anatomy-coverage"

type health struct {
	OK      bool   `json:"ok"`
	Version string `json:"version"`
}

func main() {
	home, _ := os.UserHomeDir()
	state := filepath.Join(home, ".hevy-signal")
	_ = os.MkdirAll(state, 0700)

	// One-time cleanup of persistent LaunchAgents used by older beta builds.
	cleanupLegacyAgents(home)

	// If this exact build is already running, simply reopen it.
	if portBytes, err := os.ReadFile(filepath.Join(state, "port.txt")); err == nil {
		port := string(trimSpace(portBytes))
		if port != "" {
			u := "http://127.0.0.1:" + port
			if h, ok := healthy(u); ok && h.Version == expectedVersion {
				_ = exec.Command("/usr/bin/open", u).Start()
				return
			}
		}
	}

	// Stop a stale local helper from an older non-agent build if we know its PID.
	if b, err := os.ReadFile(filepath.Join(state, "pid.txt")); err == nil {
		if pid, err := strconv.Atoi(string(trimSpace(b))); err == nil && pid > 1 {
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Signal(syscall.SIGTERM)
			}
		}
	}

	exe, err := os.Executable()
	if err != nil {
		fail("Could not locate Hevy Signal", err.Error(), state)
		return
	}
	resources := filepath.Clean(filepath.Join(filepath.Dir(exe), "..", "Resources"))
	backend := filepath.Join(resources, "HevySignalBackend")
	if _, err = os.Stat(backend); err != nil {
		fail("Hevy Signal backend is missing", err.Error(), state)
		return
	}
	_ = os.Chmod(backend, 0755)

	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fail("Could not reserve a local port", err.Error(), state)
		return
	}
	port := strconv.Itoa(probe.Addr().(*net.TCPAddr).Port)
	_ = probe.Close()

	out, _ := os.OpenFile(filepath.Join(state, "backend.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if out != nil {
		defer out.Close()
	}
	cmd := exec.Command(backend)
	cmd.Env = append(os.Environ(), "HEVY_SIGNAL_PORT="+port)
	cmd.Stdin = nil
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err = cmd.Start(); err != nil {
		fail("Hevy Signal could not start", err.Error(), state)
		return
	}

	_ = os.WriteFile(filepath.Join(state, "pid.txt"), []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0600)
	_ = os.WriteFile(filepath.Join(state, "port.txt"), []byte(port+"\n"), 0600)

	u := "http://127.0.0.1:" + port
	for i := 0; i < 80; i++ {
		if h, ok := healthy(u); ok && h.Version == expectedVersion {
			_ = exec.Command("/usr/bin/open", u).Start()
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	fail("Hevy Signal started, but the dashboard never came online", "A diagnostic log was saved in ~/.hevy-signal/backend.log", state)
}

func healthy(u string) (health, bool) {
	c := http.Client{Timeout: 400 * time.Millisecond}
	r, err := c.Get(u + "/api/health")
	if err != nil {
		return health{}, false
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return health{}, false
	}
	var h health
	if json.NewDecoder(r.Body).Decode(&h) != nil {
		return health{}, false
	}
	return h, h.OK
}

func cleanupLegacyAgents(home string) {
	uid := os.Getuid()
	domain := fmt.Sprintf("gui/%d", uid)
	labels := []string{"com.hevysignal.local.v08", "com.hevysignal.local.v09", "com.hevysignal.local.v10", "com.hevysignal.local.v11", "com.hevysignal.local.v20", "com.hevysignal.local.v21", "com.hevysignal.local.v22"}
	for _, label := range labels {
		_ = exec.Command("/bin/launchctl", "bootout", domain+"/"+label).Run()
		_ = os.Remove(filepath.Join(home, "Library", "LaunchAgents", label+".plist"))
	}
}

func trimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}

func fail(title, msg, state string) {
	_ = os.WriteFile(filepath.Join(state, "launcher.log"), []byte(title+"\n"+msg+"\n"), 0600)
	script := fmt.Sprintf(`display alert %q message %q as critical`, title, msg+"\n\nA diagnostic log was saved in ~/.hevy-signal/")
	_ = exec.Command("/usr/bin/osascript", "-e", script).Run()
}
