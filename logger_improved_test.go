package zipologger

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEmptyLogger(t *testing.T) {
	empty := EmptyLogger
	
	// Test that EmptyLogger doesn't panic and returns the message
	result := empty.Print("test message")
	if result != "test message" {
		t.Errorf("Expected 'test message', got '%s'", result)
	}
	
	result = empty.Printf("test %d", 42)
	if result != "test 42" {
		t.Errorf("Expected 'test 42', got '%s'", result)
	}
	
	result = empty.Println("hello", "world")
	expected := "hello world\n"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
	
	// Test Flush and Wait don't panic
	empty.Flush()
	empty.Wait()
}

func TestLoggerCreation(t *testing.T) {
	logger1 := NewLogger("./logs/test1.log", 1, 1, 1, false)
	logger2 := NewLogger("./logs/test1.log", 1, 1, 1, false)
	
	// Should return the same logger instance (cached)
	if logger1 != logger2 {
		t.Error("Expected same logger instance from cache")
	}
	
	logger3 := NewLogger("./logs/test2.log", 1, 1, 1, false)
	if logger1 == logger3 {
		t.Error("Expected different logger instances")
	}
}

func TestLoggerPrintFunctions(t *testing.T) {
	logger := NewLogger("./logs/test_print.log", 1, 1, 1, false)
	defer Wait()
	
	// Test Print
	result1 := logger.Print("simple message")
	if !strings.Contains(result1, "simple message") {
		t.Errorf("Print should contain message, got: %s", result1)
	}
	
	// Test Printf
	result2 := logger.Printf("formatted %s %d", "message", 42)
	if !strings.Contains(result2, "formatted message 42") {
		t.Errorf("Printf should contain formatted message, got: %s", result2)
	}
	
	// Test Println
	result3 := logger.Println("line1", "line2")
	if !strings.Contains(result3, "line1 line2") {
		t.Errorf("Println should contain joined message, got: %s", result3)
	}
	
	// Test LimitedPrintf
	logger.LimitedPrintf("test_id", time.Millisecond*10, "limited message %s", "test")
	logger.LimitedPrintf("test_id", time.Millisecond*10, "should be skipped %s", "test")
	// Can't easily test the skipping behavior without sleep, but it shouldn't panic
}

func TestLoggerConcurrency(t *testing.T) {
	logger := NewLogger("./logs/test_concurrent.log", 1, 1, 1, false)
	defer Wait()
	
	var wg sync.WaitGroup
	numGoroutines := 50
	messagesPerGoroutine := 20
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Printf("goroutine %d message %d", id, j)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify no panics occurred during concurrent logging
}

func TestLoggerFlushWait(t *testing.T) {
	logger := NewLogger("./logs/test_flush.log", 1, 1, 1, false)
	
	// Log some messages
	for i := 0; i < 10; i++ {
		logger.Printf("message %d", i)
	}
	
	// Flush should wait for all messages to be processed
	logger.Flush()
	
	// Wait again should not block indefinitely
	logger.Wait()
}

func TestGetLoggerBySuffix(t *testing.T) {
	baseName := "./logs/test_suffix"
	suffix := ".log"
	
	logger := GetLoggerBySuffix(suffix, baseName, 1, 1, 1, false)
	expectedFilename := baseName + suffix
	
	if logger.GetFileName() != expectedFilename {
		t.Errorf("Expected filename %s, got %s", expectedFilename, logger.GetFileName())
	}
}



