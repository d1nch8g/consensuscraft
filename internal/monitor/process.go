package monitor

import (
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type ProcessMonitor struct {
	allowedPIDs map[int]bool
	stopChan    chan struct{}
}

func NewProcessMonitor() *ProcessMonitor {
	return &ProcessMonitor{
		allowedPIDs: make(map[int]bool),
		stopChan:    make(chan struct{}),
	}
}

func (pm *ProcessMonitor) AllowProcess(pid int) {
	pm.allowedPIDs[pid] = true
}

func (pm *ProcessMonitor) Start() {
	go pm.monitorProcesses()
}

func (pm *ProcessMonitor) Stop() {
	close(pm.stopChan)
}

func (pm *ProcessMonitor) monitorProcesses() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.checkUnauthorizedProcesses()
		}
	}
}

func (pm *ProcessMonitor) checkUnauthorizedProcesses() {
	// Get all processes
	processes, err := pm.getAllProcesses()
	if err != nil {
		log.Printf("Failed to get processes: %v", err)
		return
	}

	for _, pid := range processes {
		if !pm.allowedPIDs[pid] && pid != os.Getpid() {
			log.Printf("Killing unauthorized process: %d", pid)
			pm.killProcess(pid)
		}
	}
}

func (pm *ProcessMonitor) getAllProcesses() ([]int, error) {
	cmd := exec.Command("ps", "-eo", "pid", "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pids []int
	var pid int
	for _, line := range string(output) {
		if line == '\n' {
			if pid > 0 {
				pids = append(pids, pid)
				pid = 0
			}
		} else if line >= '0' && line <= '9' {
			pid = pid*10 + int(line-'0')
		} else if line == ' ' || line == '\t' {
			// Skip whitespace
		}
	}
	if pid > 0 {
		pids = append(pids, pid)
	}

	return pids, nil
}

func (pm *ProcessMonitor) killProcess(pid int) {
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		log.Printf("Failed to kill process %d: %v", pid, err)
	}
}
