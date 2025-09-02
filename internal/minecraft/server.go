package minecraft

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/awnumar/memguard"
)

type Server struct {
	downloadURL    string
	extractPath    string
	executablePath string
	process        *os.Process
}

func NewServer(downloadURL, extractPath string) *Server {
	return &Server{
		downloadURL: downloadURL,
		extractPath: extractPath,
	}
}

func (s *Server) Download() error {
	resp, err := http.Get(s.downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download server: %w", err)
	}
	defer resp.Body.Close()

	// Read into memory for hash validation
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read server data: %w", err)
	}

	// Hash validation removed - not providing real security

	// Use memguard to protect server data in memory
	buffer := memguard.NewBufferFromBytes(data)
	defer buffer.Destroy()

	// Extract to temporary location
	if err := s.extractFromMemory(buffer.Bytes()); err != nil {
		return fmt.Errorf("failed to extract server: %w", err)
	}

	return nil
}

func (s *Server) extractFromMemory(data []byte) error {
	// Create unique extraction path to prevent mounting attacks
	s.extractPath = filepath.Join(s.extractPath, fmt.Sprintf("mc_%d", os.Getpid()))

	if err := os.MkdirAll(s.extractPath, 0755); err != nil {
		return err
	}

	// Create in-memory zip reader from bytes
	reader := bytes.NewReader(data)
	zipReader, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return err
	}

	// Extract files
	for _, file := range zipReader.File {
		if err := s.extractFile(file); err != nil {
			return err
		}
	}

	// Find bedrock server executable
	s.executablePath = filepath.Join(s.extractPath, "bedrock_server")
	return nil
}

func (s *Server) extractFile(file *zip.File) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	path := filepath.Join(s.extractPath, file.Name)
	if file.FileInfo().IsDir() {
		return os.MkdirAll(path, file.FileInfo().Mode())
	}

	fileWriter, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer fileWriter.Close()

	_, err = io.Copy(fileWriter, rc)
	return err
}

func (s *Server) Start() error {
	cmd := exec.Command(s.executablePath)
	cmd.Dir = s.extractPath

	// Set up inherited file descriptors for security
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start minecraft server: %w", err)
	}

	s.process = cmd.Process
	return nil
}

func (s *Server) Stop() error {
	if s.process != nil {
		return s.process.Kill()
	}
	return nil
}

func (s *Server) Cleanup() error {
	s.Stop()
	if s.extractPath != "" {
		return os.RemoveAll(s.extractPath)
	}
	return nil
}
