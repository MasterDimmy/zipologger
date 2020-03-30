package zipologger

import (
	"testing"
)

var logger *Logger

func subFunc(a string) {
	logger.Print("test from subFunc: " + a)
}

func subFunc2() {
	subFunc("c")
}

func Test_callerDepth(t *testing.T) {
	defer Wait()

	SetAlsoToStdOut(true)

	logger = NewLogger("test", 1, 1, 1)
	logger.Print("test from main")

	go func() {
		logger.Print("test from go func")
	}()

	subFunc("a")

	func() {
		logger.Print("test from func ")
		subFunc("b")
	}()

	subFunc2()
}
