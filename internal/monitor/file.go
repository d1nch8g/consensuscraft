package monitor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type FileMonitor struct {
	watchPaths   []string
	stopChan     chan struct{}
	restartFunc  func()
	checksums    map[string]string
}

func NewFileMonitor(restartFunc func()) *FileMonitor {
	return &FileMonitor{
		watchPaths:  make([]string, 0),
		stopChan:    make(chan struct{}),
		restartFunc: restartFunc,
		checksums:   make(map[string]string),
	}
}

func (fm *FileMonitor) AddPath(path string) {
	fm.watchPaths = append(fm.watchPaths, path)
	fm.calculateInitialChecksum(path)
}

func (fm *FileMonitor) Start() {
	go fm.monitorFiles()
}

func (fm *FileMonitor) Stop() {
	close(fm.stopChan)
}

func (fm *FileMonitor) monitorFiles() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-fm.stopChan:
			return
		case <-ticker.C:
			if fm.detectTampering() {
				log.Println("File tampering detected! Restarting...")
				fm.restartFunc()
				return
			}
		}
	}
}

func (fm *FileMonitor) detectTampering() bool {
	for _, path := range fm.watchPaths {
		if fm.hasFileChanged(path) {
			return true
		}
	}
	return false
}

func (fm *FileMonitor) hasFileChanged(path string) bool {
	currentChecksum := fm.calculateChecksum(path)
	originalChecksum, exists := fm.checksums[path]
	
	if !exists {
		fm.checksums[path] = currentChecksum
		return false
	}

	return currentChecksum != originalChecksum
}

func (fm *FileMonitor) calculateInitialChecksum(path string) {
	fm.checksums[path] = fm.calculateChecksum(path)
}

func (fm *FileMonitor) calculateChecksum(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "error"
	}

	// Simple checksum based on size and modification time
	return fmt.Sprintf("%d_%d", info.Size(), info.ModTime().Unix())
}

func (fm *FileMonitor) WalkAndAdd(rootPath string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fm.AddPath(path)
		}
		return nil
	})
}
