package bds

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSetup tests the constructor function
func TestNewSetup(t *testing.T) {
	t.Run("CreateNewSetup", func(t *testing.T) {
		setup := NewSetup()
		assert.NotNil(t, setup)
	})
}

// TestSetup_EnsureServer_ExistingServer tests the scenario where server already exists
func TestSetup_EnsureServer_ExistingServer(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	// Create a mock server executable
	serverPath := serverExecutable
	if runtime.GOOS == "windows" {
		serverPath = "bedrock_server.exe"
	}

	// Create the server executable
	err := os.WriteFile(serverPath, []byte("#!/bin/bash\necho 'mock server'"), 0755)
	require.NoError(t, err)

	// Test that it finds the existing server
	resultPath, err := setup.EnsureServer()
	assert.NoError(t, err)
	assert.Equal(t, serverPath, resultPath)
}

// TestSetup_EnsureServer_ExistingServerInSubdirectory tests that server in server/ subdirectory is NOT found
func TestSetup_EnsureServer_ExistingServerInSubdirectory(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	// Create server subdirectory
	serverDir := "server"
	err := os.MkdirAll(serverDir, 0755)
	require.NoError(t, err)

	// Create server executable in subdirectory
	serverPath := filepath.Join(serverDir, serverExecutable)
	err = os.WriteFile(serverPath, []byte("#!/bin/bash\necho 'mock server'"), 0755)
	require.NoError(t, err)

	// Test that it does NOT find the server in subdirectory (since we removed that logic)
	resultPath, err := setup.EnsureServer()
	assert.NoError(t, err)
	// Should not find the server in subdirectory, so it should download/extract instead
	assert.Equal(t, serverExecutable, resultPath)
}

// TestSetup_EnsureServer_ZipArchive tests extraction from existing zip archive
func TestSetup_EnsureServer_ZipArchive(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	// Create a mock zip archive with server executable
	zipPath := serverZipFile
	err := createMockServerZip(zipPath)
	require.NoError(t, err)

	// Test that it extracts and uses the server from zip
	resultPath, err := setup.EnsureServer()
	assert.NoError(t, err)
	assert.Equal(t, serverExecutable, resultPath)

	// Verify server was extracted and is executable
	_, err = os.Stat(serverExecutable)
	assert.NoError(t, err)
}

// TestSetup_EnsureServer_DownloadAndExtract tests download and extraction scenario
func TestSetup_EnsureServer_DownloadAndExtract(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	// Create a test server that serves mock zip file
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve a mock zip file
		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)
		
		// Add server executable to zip
		serverWriter, err := zipWriter.Create(serverExecutable)
		require.NoError(t, err)
		_, err = serverWriter.Write([]byte("#!/bin/bash\necho 'mock server'"))
		require.NoError(t, err)
		
		zipWriter.Close()
		w.Write(buf.Bytes())
	}))
	defer testServer.Close()

	// Override the download URL to use our test server
	originalURL := serverDownloadURL
	serverDownloadURL = testServer.URL
	defer func() { serverDownloadURL = originalURL }()

	// Test that it downloads and extracts the server
	resultPath, err := setup.EnsureServer()
	assert.NoError(t, err)
	assert.Equal(t, serverExecutable, resultPath)

	// Verify server was downloaded and extracted
	_, err = os.Stat(serverExecutable)
	assert.NoError(t, err)
}

// TestSetup_EnsureServer_DownloadError tests error handling for download failures
func TestSetup_EnsureServer_DownloadError(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	// Create a test server that returns error
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer testServer.Close()

	// Override the download URL to use our test server
	originalURL := serverDownloadURL
	serverDownloadURL = testServer.URL
	defer func() { serverDownloadURL = originalURL }()

	// Test that it returns an error for failed download
	resultPath, err := setup.EnsureServer()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download failed")
	assert.Empty(t, resultPath)
}

// TestSetup_checkCurrentDirectory tests the directory checking function
func TestSetup_checkCurrentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	t.Run("NoServerExists", func(t *testing.T) {
		result := setup.checkCurrentDirectory()
		assert.Empty(t, result)
	})

	t.Run("ServerExists", func(t *testing.T) {
		// Create server executable
		err := os.WriteFile(serverExecutable, []byte("mock"), 0755)
		require.NoError(t, err)

		result := setup.checkCurrentDirectory()
		assert.Equal(t, serverExecutable, result)
	})

	t.Run("ServerExistsWithAlternativeName", func(t *testing.T) {
		// Test with alternative executable name
		altName := "bedrock_server"
		if runtime.GOOS == "windows" {
			altName = "bedrock_server.exe"
		}

		err := os.WriteFile(altName, []byte("mock"), 0755)
		require.NoError(t, err)

		result := setup.checkCurrentDirectory()
		assert.Equal(t, altName, result)
	})
}

// TestSetup_checkZipArchive tests the zip archive checking function
func TestSetup_checkZipArchive(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	t.Run("NoZipExists", func(t *testing.T) {
		result := setup.checkZipArchive()
		assert.Empty(t, result)
	})

	t.Run("SpecificZipExists", func(t *testing.T) {
		// Create specific zip file
		err := os.WriteFile(serverZipFile, []byte("mock zip"), 0644)
		require.NoError(t, err)

		result := setup.checkZipArchive()
		assert.Equal(t, serverZipFile, result)
	})

	t.Run("WildcardZipExists", func(t *testing.T) {
		// Remove any existing specific zip file first
		os.Remove(serverZipFile)
		
		// Create wildcard matching zip file (different version)
		wildcardZip := "bedrock-server-1.20.0.0.zip"
		err := os.WriteFile(wildcardZip, []byte("mock zip"), 0644)
		require.NoError(t, err)

		result := setup.checkZipArchive()
		// Should return the wildcard zip since it's the only one that exists
		// Note: The function checks for specific zip first, then wildcard
		assert.Equal(t, wildcardZip, result)
	})
}

// TestSetup_downloadServerZip tests the download functionality
func TestSetup_downloadServerZip(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	t.Run("SuccessfulDownload", func(t *testing.T) {
		// Create a test server that serves mock zip file
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("mock zip content"))
		}))
		defer testServer.Close()

		// Override the download URL
		originalURL := serverDownloadURL
		serverDownloadURL = testServer.URL
		defer func() { serverDownloadURL = originalURL }()

		err := setup.downloadServerZip()
		assert.NoError(t, err)

		// Verify file was downloaded
		_, err = os.Stat(serverZipFile)
		assert.NoError(t, err)
	})

	t.Run("DownloadError", func(t *testing.T) {
		// Create a test server that returns error
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not Found", http.StatusNotFound)
		}))
		defer testServer.Close()

		// Override the download URL
		originalURL := serverDownloadURL
		serverDownloadURL = testServer.URL
		defer func() { serverDownloadURL = originalURL }()

		err := setup.downloadServerZip()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "download failed")
	})
}

// TestSetup_extractServer tests the extraction functionality
func TestSetup_extractServer(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	t.Run("SuccessfulExtraction", func(t *testing.T) {
		// Create a mock zip archive
		err := createMockServerZip(serverZipFile)
		require.NoError(t, err)

		err = setup.extractServer()
		assert.NoError(t, err)

		// Verify files were extracted
		_, err = os.Stat(serverExecutable)
		assert.NoError(t, err)

		// Verify executable permissions on Unix-like systems
		if runtime.GOOS != "windows" {
			info, err := os.Stat(serverExecutable)
			assert.NoError(t, err)
			assert.NotZero(t, info.Mode()&0111) // Check if executable bit is set
		}
	})

	t.Run("ExtractionError_NoZipFile", func(t *testing.T) {
		// Remove any existing zip files to ensure no zip is found
		os.Remove(serverZipFile)
		os.Remove("bedrock-server-1.20.0.0.zip")
		
		err := setup.extractServer()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no zip file found")
	})

	t.Run("ExtractionError_InvalidZip", func(t *testing.T) {
		// Create invalid zip file
		err := os.WriteFile(serverZipFile, []byte("invalid zip content"), 0644)
		require.NoError(t, err)

		err = setup.extractServer()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open zip file")
	})
}

// TestSetup_extractFile tests individual file extraction
func TestSetup_extractFile(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	setup := NewSetup()

	// Create a mock zip file
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	
	fileWriter, err := zipWriter.Create("test_file.txt")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("test content"))
	require.NoError(t, err)
	
	zipWriter.Close()

	// Write zip to file
	zipFile := "test.zip"
	err = os.WriteFile(zipFile, buf.Bytes(), 0644)
	require.NoError(t, err)

	// Open zip file
	reader, err := zip.OpenReader(zipFile)
	require.NoError(t, err)
	defer reader.Close()

	// Extract the file
	file := reader.File[0]
	err = setup.extractFile(file, "extracted_test_file.txt")
	assert.NoError(t, err)

	// Verify file was extracted
	content, err := os.ReadFile("extracted_test_file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

// TestSetup_Integration_AllScenarios tests integration of all scenarios
func TestSetup_Integration_AllScenarios(t *testing.T) {
	t.Run("Scenario1_ExistingServer", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		setup := NewSetup()

		// Create existing server
		err := os.WriteFile(serverExecutable, []byte("mock server"), 0755)
		require.NoError(t, err)

		resultPath, err := setup.EnsureServer()
		assert.NoError(t, err)
		assert.Equal(t, serverExecutable, resultPath)
	})

	t.Run("Scenario2_ZipArchive", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		setup := NewSetup()

		// Create zip archive
		err := createMockServerZip(serverZipFile)
		require.NoError(t, err)

		resultPath, err := setup.EnsureServer()
		assert.NoError(t, err)
		assert.Equal(t, serverExecutable, resultPath)
	})

	t.Run("Scenario3_DownloadRequired", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		setup := NewSetup()

		// Create test server for download
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var buf bytes.Buffer
			zipWriter := zip.NewWriter(&buf)
			
			serverWriter, err := zipWriter.Create(serverExecutable)
			require.NoError(t, err)
			_, err = serverWriter.Write([]byte("mock server"))
			require.NoError(t, err)
			
			zipWriter.Close()
			w.Write(buf.Bytes())
		}))
		defer testServer.Close()

		// Override download URL
		originalURL := serverDownloadURL
		serverDownloadURL = testServer.URL
		defer func() { serverDownloadURL = originalURL }()

		resultPath, err := setup.EnsureServer()
		assert.NoError(t, err)
		assert.Equal(t, serverExecutable, resultPath)
	})
}

// Helper function to create a mock server zip file
func createMockServerZip(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Add server executable
	serverWriter, err := zipWriter.Create(serverExecutable)
	if err != nil {
		return err
	}
	_, err = serverWriter.Write([]byte("#!/bin/bash\necho 'mock server'"))
	if err != nil {
		return err
	}

	// Add some additional files to simulate real server zip
	files := []string{
		"server.properties",
		"whitelist.json",
		"permissions.json",
	}

	for _, filename := range files {
		writer, err := zipWriter.Create(filename)
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte("mock content for " + filename))
		if err != nil {
			return err
		}
	}

	return nil
}
