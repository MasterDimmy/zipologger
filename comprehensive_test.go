package zipologger

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestComprehensiveLoggerUsage(t *testing.T) {
	// Test 1: Basic logger functionality
	t.Run("BasicLogger", func(t *testing.T) {
		logger := NewLogger("./logs/comprehensive_basic.log", 1, 1, 1, false)
		logger.Print("basic print")
		logger.Printf("basic printf %d", 42)
		logger.Println("basic println", "test")
		
		Wait()
		
		content, err := os.ReadFile("./logs/comprehensive_basic.log")
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}
		
		expected := []string{"basic print", "basic printf 42", "basic println test"}
		for _, msg := range expected {
			if !strings.Contains(string(content), msg) {
				t.Errorf("Missing message: %s", msg)
			}
		}
	})

	// Test 2: EmptyLogger - should not create any files or perform operations
	t.Run("EmptyLogger", func(t *testing.T) {
		empty := EmptyLogger
		
		// These should not panic and should return the input
		result1 := empty.Print("empty print")
		result2 := empty.Printf("empty printf %d", 42)
		result3 := empty.Println("empty println", "test")
		
		if result1 != "empty print" {
			t.Errorf("Expected 'empty print', got '%s'", result1)
		}
		if result2 != "empty printf 42" {
			t.Errorf("Expected 'empty printf 42', got '%s'", result2)
		}
		if !strings.Contains(result3, "empty println test") {
			t.Errorf("Expected message containing 'empty println test', got '%s'", result3)
		}
		
		// EmptyLogger should not create any files
		empty.Flush()
		empty.Wait()
		
		// Verify no files were created by EmptyLogger
		// (This is implicit - if it worked without error, it's correct)
	})

	// Test 3: Multiple loggers with different configurations
	t.Run("MultipleLoggers", func(t *testing.T) {
		logger1 := NewLogger("./logs/comprehensive_multi1.log", 2, 2, 2, true)
		logger2 := NewLogger("./logs/comprehensive_multi2.log", 1, 1, 1, false)
		
		logger1.Print("logger1 message")
		logger2.Print("logger2 message")
		
		// Test Flush on individual loggers
		logger1.Flush()
		logger2.Flush()
		
		Wait()
		
		content1, err1 := os.ReadFile("./logs/comprehensive_multi1.log")
		if err1 != nil {
			t.Fatalf("Failed to read multi1 log file: %v", err1)
		}
		
		content2, err2 := os.ReadFile("./logs/comprehensive_multi2.log")
		if err2 != nil {
			t.Fatalf("Failed to read multi2 log file: %v", err2)
		}
		
		if !strings.Contains(string(content1), "logger1 message") {
			t.Error("multi1 log missing message")
		}
		if !strings.Contains(string(content2), "logger2 message") {
			t.Error("multi2 log missing message")
		}
	})

	// Test 4: Concurrent logging
	t.Run("ConcurrentLogging", func(t *testing.T) {
		logger := NewLogger("./logs/comprehensive_concurrent.log", 1, 1, 1, false)
		
		var wg sync.WaitGroup
		numGoroutines := 10
		messagesPerGoroutine := 5
		
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					logger.Printf("goroutine %d message %d", goroutineID, j)
				}
			}(i)
		}
		
		wg.Wait()
		Wait()
		
		content, err := os.ReadFile("./logs/comprehensive_concurrent.log")
		if err != nil {
			t.Fatalf("Failed to read concurrent log file: %v", err)
		}
		
		totalMessages := numGoroutines * messagesPerGoroutine
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != totalMessages {
			t.Errorf("Expected %d messages, got %d lines", totalMessages, len(lines))
		}
	})

	// Test 5: Logger with subdirectories
	t.Run("SubdirectoryLogging", func(t *testing.T) {
		logger := NewLogger("./logs/nested/deep/test_subdir.log", 1, 1, 1, false)
		logger.Print("subdirectory test")
		
		Wait()
		
		content, err := os.ReadFile("./logs/nested/deep/test_subdir.log")
		if err != nil {
			t.Fatalf("Failed to read subdirectory log file: %v", err)
		}
		
		if !strings.Contains(string(content), "subdirectory test") {
			t.Error("Subdirectory log missing message")
		}
	})

	// Test 6: LimitedPrintf functionality
	t.Run("LimitedPrintf", func(t *testing.T) {
		logger := NewLogger("./logs/comprehensive_limited.log", 1, 1, 1, false)
		
		// Should log the first call
		logger.LimitedPrintf("test_key", time.Millisecond*100, "limited message %d", 1)
		
		// Should skip this call (within duration)
		logger.LimitedPrintf("test_key", time.Millisecond*100, "limited message %d", 2)
		
		// Wait for duration to expire
		time.Sleep(time.Millisecond * 150)
		
		// Should log this call (duration expired)
		logger.LimitedPrintf("test_key", time.Millisecond*100, "limited message %d", 3)
		
		Wait()
		
		content, err := os.ReadFile("./logs/comprehensive_limited.log")
		if err != nil {
			t.Fatalf("Failed to read limited log file: %v", err)
		}
		
		// Should contain message 1 and 3, but not 2
		if !strings.Contains(string(content), "limited message 1") {
			t.Error("Missing first limited message")
		}
		if strings.Contains(string(content), "limited message 2") {
			t.Error("Should not contain second limited message")
		}
		if !strings.Contains(string(content), "limited message 3") {
			t.Error("Missing third limited message")
		}
	})

	// Test 7: Defer Wait pattern (like user examples)
	t.Run("DeferWaitPattern", func(t *testing.T) {
		func() {
			defer Wait()
			
			logger := NewLogger("./logs/comprehensive_defer.log", 1, 1, 1, false)
			logger.Print("defer wait test message")
			
			// Message should be written even though Wait() is called in defer
		}()
		
		// File should exist after function returns
		content, err := os.ReadFile("./logs/comprehensive_defer.log")
		if err != nil {
			t.Fatalf("Failed to read defer log file: %v", err)
		}
		
		if !strings.Contains(string(content), "defer wait test message") {
			t.Error("Defer wait log missing message")
		}
	})

	// Test 8: Double Wait and Flush pattern
	t.Run("DoubleWaitFlush", func(t *testing.T) {
		logger := NewLogger("./logs/comprehensive_double.log", 1, 1, 1, false)
		
		logger.Print("first message")
		logger.Flush()
		logger.Wait()
		
		logger.Print("second message")
		logger.Flush()
		logger.Wait()
		
		// Verify file contains both messages
		content, err := os.ReadFile("./logs/comprehensive_double.log")
		if err != nil {
			t.Fatalf("Failed to read double log file: %v", err)
		}

		if !strings.Contains(string(content), "first message") {
			t.Error("Missing first message")
		}
		if !strings.Contains(string(content), "second message") {
			t.Error("Missing second message")
		}
	})
}

func TestLoggerEdgeCases(t *testing.T) {
	// Test empty Println
	t.Run("EmptyPrintln", func(t *testing.T) {
		logger := NewLogger("./logs/comprehensive_empty_println.log", 1, 1, 1, false)
		result := logger.Println()
		if result != "" {
			t.Errorf("Expected empty string for empty Println, got '%s'", result)
		}
		
		Wait()
		
		// The key test is that result is empty string - file creation is not guaranteed for empty Println
		// This matches the original library behavior
	})

	// Test very large messages
	t.Run("LargeMessages", func(t *testing.T) {
		logger := NewLogger("./logs/comprehensive_large.log", 10, 5, 7, false)
		largeMessage := strings.Repeat("x", 10000)
		logger.Print(largeMessage)
		
		Wait()
		
		content, err := os.ReadFile("./logs/comprehensive_large.log")
		if err != nil {
			t.Fatalf("Failed to read large log file: %v", err)
		}
		
		if !strings.Contains(string(content), largeMessage) {
			t.Error("Large message not properly logged")
		}
	})
}

func cleanupComprehensiveTestFiles() {
	filesToRemove := []string{
		"./logs/comprehensive_basic.log",
		"./logs/comprehensive_multi1.log", 
		"./logs/comprehensive_multi2.log",
		"./logs/comprehensive_concurrent.log",
		"./logs/comprehensive_limited.log",
		"./logs/comprehensive_defer.log",
		"./logs/comprehensive_double.log",
		"./logs/comprehensive_empty_println.log",
		"./logs/comprehensive_large.log",
		"./logs/nested/deep/test_subdir.log",
	}
	
	for _, file := range filesToRemove {
		os.Remove(file)
	}
	
	// Remove directories
	os.RemoveAll("./logs/nested")
	os.RemoveAll("./logs")
}

// TestMain ensures clean state before and after tests
func TestMain(m *testing.M) {
	// Clean up any existing test files
	cleanupComprehensiveTestFiles()
	cleanupTestFiles() // from usage_test.go
	
	// Run tests
	code := m.Run()
	
	// Clean up after tests
	cleanupComprehensiveTestFiles()
	cleanupTestFiles()
	
	os.Exit(code)
}