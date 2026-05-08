package zipologger

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkLoggerPrint(b *testing.B) {
	logger := NewLogger("bench_print.log", 10, 5, 7, false)
	defer func() {
		Wait()
	}()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Print("test message")
	}
}

func BenchmarkEmptyLoggerPrint(b *testing.B) {
	logger := EmptyLogger
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Print("test message")
	}
}

func BenchmarkLoggerPrintf(b *testing.B) {
	logger := NewLogger("bench_printf.log", 10, 5, 7, false)
	defer func() {
		Wait()
	}()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Printf("test message %d", i)
	}
}

func BenchmarkEmptyLoggerPrintf(b *testing.B) {
	logger := EmptyLogger
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Printf("test message %d", i)
	}
}

func BenchmarkLoggerPrintln(b *testing.B) {
	logger := NewLogger("bench_println.log", 10, 5, 7, false)
	defer func() {
		Wait()
	}()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Println("test", "message", i)
	}
}

func BenchmarkEmptyLoggerPrintln(b *testing.B) {
	logger := EmptyLogger
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Println("test", "message", i)
	}
}

func BenchmarkLoggerLimitedPrintf(b *testing.B) {
	logger := NewLogger("bench_limited.log", 10, 5, 7, false)
	defer func() {
		Wait()
	}()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LimitedPrintf("test", time.Second, "test message %d", i)
	}
}

func BenchmarkEmptyLoggerLimitedPrintf(b *testing.B) {
	logger := EmptyLogger
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LimitedPrintf("test", time.Second, "test message %d", i)
	}
}