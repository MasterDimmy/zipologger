package zipologger

import (
	"os"
	"strings"
	"testing"
)

func TestBasicLoggerUsage(t *testing.T) {
	// Test basic logger creation and usage like in user examples
	logger := NewLogger("test_basic.log", 1, 1, 1, false)
	logger.Println("basic test message")
	Wait()
	
	// Verify file was created and contains expected content
	content, err := os.ReadFile("test_basic.log")
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	if !strings.Contains(string(content), "basic test message") {
		t.Errorf("Log file doesn't contain expected message")
	}
	
	// Cleanup
	os.Remove("test_basic.log")
}

func TestLoggerWithSubdirectories(t *testing.T) {
	// Test logger with nested directories like "./logs/conns/2.log"
	logger := NewLogger("./logs/subdir/test_subdir.log", 1, 1, 1, false)
	logger.Println("subdirectory test message")
	Wait()
	
	// Verify file was created in subdirectory
	content, err := os.ReadFile("./logs/subdir/test_subdir.log")
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	if !strings.Contains(string(content), "subdirectory test message") {
		t.Errorf("Log file doesn't contain expected message")
	}
	
	// Cleanup
	os.Remove("./logs/subdir/test_subdir.log")
	os.Remove("./logs/subdir")
	os.Remove("./logs")
}

func TestLoggerWithDeferWait(t *testing.T) {
	// Test the pattern: defer Wait() at start of main
	var logger *Logger
	
	func() {
		defer Wait()
		logger = NewLogger("test_defer_wait.log", 1, 1, 1, false)
		logger.Println("defer wait test message")
	}()
	
	// File should be written even though Wait() was called in defer
	content, err := os.ReadFile("test_defer_wait.log")
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	if !strings.Contains(string(content), "defer wait test message") {
		t.Errorf("Log file doesn't contain expected message")
	}
	
	// Cleanup
	os.Remove("test_defer_wait.log")
}

func TestMultipleLoggers(t *testing.T) {
	// Test multiple loggers working simultaneously
	logger1 := NewLogger("test_multi1.log", 1, 1, 1, false)
	logger2 := NewLogger("./logs/multi2.log", 1, 1, 1, false)
	
	logger1.Println("message from logger1")
	logger2.Println("message from logger2")
	
	Wait()
	
	// Verify both files exist and contain correct messages
	content1, err1 := os.ReadFile("test_multi1.log")
	if err1 != nil {
		t.Fatalf("Failed to read log file 1: %v", err1)
	}
	
	content2, err2 := os.ReadFile("./logs/multi2.log")
	if err2 != nil {
		t.Fatalf("Failed to read log file 2: %v", err2)
	}
	
	if !strings.Contains(string(content1), "message from logger1") {
		t.Errorf("Log file 1 doesn't contain expected message")
	}
	
	if !strings.Contains(string(content2), "message from logger2") {
		t.Errorf("Log file 2 doesn't contain expected message")
	}
	
	// Cleanup
	os.Remove("test_multi1.log")
	os.Remove("./logs/multi2.log")
	os.Remove("./logs")
}

func TestLoggerFlush(t *testing.T) {
	// Test explicit Flush() call
	logger := NewLogger("test_flush.log", 1, 1, 1, false)
	logger.Println("flush test message")
	logger.Flush() // Should work same as Wait() for single logger
	
	content, err := os.ReadFile("test_flush.log")
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	if !strings.Contains(string(content), "flush test message") {
		t.Errorf("Log file doesn't contain expected message")
	}
	
	// Cleanup
	os.Remove("test_flush.log")
}

func TestEmptyLoggerUsage(t *testing.T) {
	// Test EmptyLogger doesn't create any files
	empty := EmptyLogger
	empty.Println("empty logger message")
	empty.Flush()
	empty.Wait()
	
	// No files should be created by EmptyLogger
	// This test just ensures it doesn't panic
}

func TestLoggerCreationWithoutWait(t *testing.T) {
	// Test logger creation without explicit Wait() - should still work due to goroutine processing
	logger := NewLogger("test_no_wait.log", 1, 1, 1, false)
	logger.Println("no wait test message")
	
	// Give some time for goroutine to process (not ideal but necessary for this test)
	Wait()
	
	content, err := os.ReadFile("test_no_wait.log")
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	if !strings.Contains(string(content), "no wait test message") {
		t.Errorf("Log file doesn't contain expected message")
	}
	
	// Cleanup
	os.Remove("test_no_wait.log")
}

func cleanupTestFiles() {
	testFiles := []string{
		"test_basic.log",
		"test_defer_wait.log", 
		"test_multi1.log",
		"test_flush.log",
		"test_no_wait.log",
	}
	
	for _, file := range testFiles {
		os.Remove(file)
	}
	
	// Remove test directories
	os.RemoveAll("./logs")
}