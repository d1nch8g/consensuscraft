package logger

import (
	"bytes"
	"log"
	"regexp"
	"strings"
	"testing"
	"time"
)

// captureOutput captures log output for testing
func captureOutput(fn func()) string {
	var buf bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&buf)
	fn()
	log.SetOutput(originalOutput) // Reset to original
	return buf.String()
}

// TestFormatMessage tests the internal formatMessage function
func TestFormatMessage(t *testing.T) {
	message := "test message"
	level := "INFO"
	
	result := formatMessage(level, message)
	
	// Check that the format matches the expected pattern
	// [2025-09-25 19:04:40:650 INFO] [CONSENSUSCRAFT] test message
	pattern := `^\[20\d{2}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}:\d{3} INFO\] \[CONSENSUSCRAFT\] test message$`
	matched, err := regexp.MatchString(pattern, result)
	
	if err != nil {
		t.Fatalf("Regex error: %v", err)
	}
	
	if !matched {
		t.Errorf("Format doesn't match expected pattern. Got: %s", result)
	}
	
	// Check that it contains all required components
	if !strings.Contains(result, "[INFO]") {
		t.Error("Missing log level [INFO]")
	}
	
	if !strings.Contains(result, "[CONSENSUSCRAFT]") {
		t.Error("Missing [CONSENSUSCRAFT] identifier")
	}
	
	if !strings.Contains(result, "test message") {
		t.Error("Missing actual message")
	}
}

// TestInfoLogging tests Info and Infof functions
func TestInfoLogging(t *testing.T) {
	// Test Info function
	output := captureOutput(func() {
		Info("info message")
	})
	
	if !strings.Contains(output, "[INFO]") {
		t.Error("Info() should contain [INFO] level")
	}
	
	if !strings.Contains(output, "info message") {
		t.Error("Info() should contain the message")
	}
	
	// Test Infof function
	output = captureOutput(func() {
		Infof("formatted %s %d", "message", 42)
	})
	
	if !strings.Contains(output, "[INFO]") {
		t.Error("Infof() should contain [INFO] level")
	}
	
	if !strings.Contains(output, "formatted message 42") {
		t.Error("Infof() should contain the formatted message")
	}
}

// TestErrorLogging tests Error and Errorf functions
func TestErrorLogging(t *testing.T) {
	// Test Error function
	output := captureOutput(func() {
		Error("error message")
	})
	
	if !strings.Contains(output, "[ERROR]") {
		t.Error("Error() should contain [ERROR] level")
	}
	
	if !strings.Contains(output, "error message") {
		t.Error("Error() should contain the message")
	}
	
	// Test Errorf function
	output = captureOutput(func() {
		Errorf("error code: %d", 500)
	})
	
	if !strings.Contains(output, "[ERROR]") {
		t.Error("Errorf() should contain [ERROR] level")
	}
	
	if !strings.Contains(output, "error code: 500") {
		t.Error("Errorf() should contain the formatted message")
	}
}

// TestWarnLogging tests Warn and Warnf functions
func TestWarnLogging(t *testing.T) {
	// Test Warn function
	output := captureOutput(func() {
		Warn("warning message")
	})
	
	if !strings.Contains(output, "[WARN]") {
		t.Error("Warn() should contain [WARN] level")
	}
	
	if !strings.Contains(output, "warning message") {
		t.Error("Warn() should contain the message")
	}
	
	// Test Warnf function
	output = captureOutput(func() {
		Warnf("warning: %s", "deprecated")
	})
	
	if !strings.Contains(output, "[WARN]") {
		t.Error("Warnf() should contain [WARN] level")
	}
	
	if !strings.Contains(output, "warning: deprecated") {
		t.Error("Warnf() should contain the formatted message")
	}
}

// TestDebugLogging tests Debug and Debugf functions
func TestDebugLogging(t *testing.T) {
	// Test Debug function
	output := captureOutput(func() {
		Debug("debug message")
	})
	
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("Debug() should contain [DEBUG] level")
	}
	
	if !strings.Contains(output, "debug message") {
		t.Error("Debug() should contain the message")
	}
	
	// Test Debugf function
	output = captureOutput(func() {
		Debugf("debug value: %v", true)
	})
	
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("Debugf() should contain [DEBUG] level")
	}
	
	if !strings.Contains(output, "debug value: true") {
		t.Error("Debugf() should contain the formatted message")
	}
}

// TestLegacyFunctions tests backward compatibility functions
func TestLegacyFunctions(t *testing.T) {
	// Test Print function (should default to INFO)
	output := captureOutput(func() {
		Print("legacy print")
	})
	
	if !strings.Contains(output, "[INFO]") {
		t.Error("Print() should default to [INFO] level")
	}
	
	if !strings.Contains(output, "legacy print") {
		t.Error("Print() should contain the message")
	}
	
	// Test Printf function (should default to INFO)
	output = captureOutput(func() {
		Printf("legacy %s", "printf")
	})
	
	if !strings.Contains(output, "[INFO]") {
		t.Error("Printf() should default to [INFO] level")
	}
	
	if !strings.Contains(output, "legacy printf") {
		t.Error("Printf() should contain the formatted message")
	}
	
	// Test Println function (should default to INFO)
	output = captureOutput(func() {
		Println("legacy println")
	})
	
	if !strings.Contains(output, "[INFO]") {
		t.Error("Println() should default to [INFO] level")
	}
	
	if !strings.Contains(output, "legacy println") {
		t.Error("Println() should contain the message")
	}
}

// TestTimestampFormat tests that timestamps are in the correct format
func TestTimestampFormat(t *testing.T) {
	output := captureOutput(func() {
		Info("timestamp test")
	})
	
	// Extract timestamp from output using regex
	// Pattern: [2025-09-25 19:04:40:650 INFO]
	timestampPattern := `\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}:\d{3}) INFO\]`
	re := regexp.MustCompile(timestampPattern)
	matches := re.FindStringSubmatch(output)
	
	if len(matches) < 2 {
		t.Fatalf("Could not extract timestamp from output: %s", output)
	}
	
	timestampStr := matches[1]
	
	// Parse the timestamp to ensure it's valid (note: Go uses different format for milliseconds)
	_, err := time.Parse("2006-01-02 15:04:05.000", timestampStr[:19]+"."+timestampStr[20:])
	if err != nil {
		t.Errorf("Invalid timestamp format: %s, error: %v", timestampStr, err)
	}
	
	// Check that timestamp is recent (within last few seconds)
	parsedTime, _ := time.Parse("2006-01-02 15:04:05", timestampStr[:19])
	now := time.Now()
	diff := now.Sub(parsedTime)
	
	if diff > 5*time.Second || diff < -1*time.Second {
		t.Errorf("Timestamp seems incorrect. Parsed: %v, Now: %v, Diff: %v", parsedTime, now, diff)
	}
}

// TestConsensuscraftIdentifier tests that all log messages contain the CONSENSUSCRAFT identifier
func TestConsensuscraftIdentifier(t *testing.T) {
	testCases := []func(){
		func() { Info("test") },
		func() { Error("test") },
		func() { Warn("test") },
		func() { Debug("test") },
		func() { Print("test") },
		func() { Printf("test") },
		func() { Println("test") },
	}
	
	for i, testCase := range testCases {
		output := captureOutput(testCase)
		
		if !strings.Contains(output, "[CONSENSUSCRAFT]") {
			t.Errorf("Test case %d: Missing [CONSENSUSCRAFT] identifier in output: %s", i, output)
		}
	}
}

// TestMultipleArguments tests functions with multiple arguments
func TestMultipleArguments(t *testing.T) {
	output := captureOutput(func() {
		Info("multiple", "arguments", 123, true)
	})
	
	// fmt.Sprint concatenates without spaces, so "multiplearguments123 true" is expected
	if !strings.Contains(output, "multiplearguments123 true") {
		t.Errorf("Multiple arguments not properly formatted: %s", output)
	}
}

// TestEmptyMessage tests logging with empty messages
func TestEmptyMessage(t *testing.T) {
	output := captureOutput(func() {
		Info("")
	})
	
	if !strings.Contains(output, "[INFO]") {
		t.Error("Empty message should still contain log level")
	}
	
	if !strings.Contains(output, "[CONSENSUSCRAFT]") {
		t.Error("Empty message should still contain CONSENSUSCRAFT identifier")
	}
}
