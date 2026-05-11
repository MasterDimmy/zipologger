package zipologger

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func subFunc(a string, logger *Logger) {
	logger.Print("test from subFunc: " + a)
}

func subFunc2(logger *Logger) {
	subFunc("c", logger)
}

func Test_callerDepth(t *testing.T) {
	defer Wait()

	SetAlsoToStdout(true)

	logger := NewLogger("./logs/test_caller.log", 1, 1, 1, true)
	logger.Print("test from main")

	done := make(chan bool)
	go func() {
		logger.Print("test from go func")
		done <- true
	}()

	subFunc("a", logger)

	func() {
		logger.Print("test from func ")
		subFunc("b", logger)
	}()

	subFunc2(logger)

	// Wait for goroutine to complete
	<-done

	Wait() // Ensure all messages are processed

	// Verify log file contains expected content
	content, err := os.ReadFile("./logs/test_caller.log")
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	expectedMessages := []string{
		"test from main",
		"test from go func", 
		"test from subFunc: a",
		"test from func",
		"test from subFunc: b",
		"test from subFunc: c",
	}

	for _, expected := range expectedMessages {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Log file missing expected message: %s", expected)
		}
	}
}

func Test_println(t *testing.T) {
	defer Wait()

	SetAlsoToStdout(true)

	logger := NewLogger("./logs/test_println.log", 1, 1, 1, true)

	logger.Println("1")

	logger.Println("1", "2")

	d := 4
	logger.Println("1", 3)

	logger.Println(3, "2", d)

	t.Log("wait 1")
	Wait()
	
	// Add another log entry between Wait() calls
	logger.Println("message between waits")

	t.Log("wait 2")  
	Wait()

	// Verify all messages were written to log file after both Wait() calls
	content, err := os.ReadFile("./logs/test_println.log")
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	expectedMessages := []string{
		"1",
		"1 2", 
		"1 3",
		"3 2 4",
		"message between waits",
	}

	for _, expected := range expectedMessages {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Log file missing expected message: %s", expected)
		}
	}
}

func Test_2logger_by_suffix(t *testing.T) {
	defer Wait()

	SetAlsoToStdout(false)
	os.MkdirAll("./logs", 0755)

	for i := 0; i < 10; i++ {
		l1 := GetLoggerBySuffix("a.log", "./logs/test_", 1, 1, 1, false)
		l2 := GetLoggerBySuffix("b.log", "./logs/test_", 1, 1, 1, false)

		st := l2.Printf("%d aaaa - %s", i, time.Now().String())
		if i%5 == 0 {
			t.Log(st)
		}

		l2.Flush()

		l1.Print("bbb")

		l1.Flush()
		l2.Flush()
		l1.Wait()
		l2.Wait()
		l1.Wait()
		l1.Wait()
	}

	Wait()

	// Verify log files contain expected content
	content1, err1 := os.ReadFile("./logs/test_a.log")
	if err1 != nil {
		t.Fatalf("Failed to read test_a.log: %v", err1)
	}

	content2, err2 := os.ReadFile("./logs/test_b.log")  
	if err2 != nil {
		t.Fatalf("Failed to read test_b.log: %v", err2)
	}

	if !strings.Contains(string(content1), "bbb") {
		t.Error("test_a.log missing 'bbb' messages")
	}

	if string(content1) == "" || string(content2) == "" {
		t.Error("Log files are empty")
	}
}

func Test_CloseFiles(t *testing.T) {
	defer Wait()

	os.MkdirAll("./logs", 0755)
	sw := GetLoggerBySuffix(fmt.Sprintf("123123.log"), "./logs/test_", 1, 1, 1, false)
	sw.Print("123123123")

	Wait()
	Wait()

	// Verify first log file
	content, err := os.ReadFile("./logs/test_123123.log")
	if err != nil {
		t.Fatalf("Failed to read test_123123.log: %v", err)
	}
	if !strings.Contains(string(content), "123123123") {
		t.Error("test_123123.log missing expected content")
	}

	SetAlsoToStdout(false)

	for i := 0; i < 5; i++ { // Reduced iterations for testing
		l1 := GetLoggerBySuffix(fmt.Sprintf("a_%d.log", i), "./logs/test_", 1, 1, 1, false)
		l1.Printf("%d a", i)
	}

	Wait()
	Wait()

	// Verify at least one of the generated files
	content2, err2 := os.ReadFile("./logs/test_a_0.log")
	if err2 != nil {
		t.Fatalf("Failed to read test_a_0.log: %v", err2)
	}
	if !strings.Contains(string(content2), "0 a") {
		t.Error("test_a_0.log missing expected content")
	}
}
