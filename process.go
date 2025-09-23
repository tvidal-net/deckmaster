package main

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	childProcessCGroup = createNewCGroup("deckmaster.scope")
)

func runningCGroup() string {
	s, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		errorLogF("Unable to read the cgroup for the current process")
		panic(err)
	}
	split := strings.Split(string(s), ":")
	cgroup := split[len(split)-1]
	return strings.TrimSpace(cgroup)
}

func createNewCGroup(name string) string {
	parent := filepath.Dir(runningCGroup())
	cgroupPath := path.Join("/sys/fs/cgroup", parent, name)
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		errorLogF("Unable to create new cgroup for child processes\n\t", cgroupPath)
		panic(err)
	}
	return cgroupPath
}

func moveProcessToCGroup(pid int, cgroup string) error {
	cgroupFile := path.Join(cgroup, "cgroup.procs")
	return os.WriteFile(cgroupFile, []byte(strconv.Itoa(pid)), 0644)
}

func expandExecutable(exe string) string {
	for _, base := range PATH {
		cmd := filepath.Join(base, exe)
		s, e := os.Stat(cmd)
		if e != nil || s.IsDir() {
			continue
		}
		fileMode := s.Mode()
		if fileMode&0111 != 0 {
			return cmd
		}
	}
	return exe
}

// executes a command.
func executeCommand(cmd string) error {
	args := SPACES.Split(cmd, -1)
	exe := expandExecutable(args[0])

	command := exec.Command(exe, args[1:]...)
	if err := command.Start(); err != nil {
		errorLogF("failed to execute '%s %s'", exe, args[1:])
		return err
	}
	pid := command.Process.Pid
	if err := moveProcessToCGroup(pid, childProcessCGroup); err != nil {
		errorLog(err, "Unable to move child process %d to cgroup", pid)
	}
	return command.Process.Release()
}
