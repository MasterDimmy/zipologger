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

	SetAlsoToStdout(true)

	logger = NewLogger("test", 1, 1, 1, true)
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

func Test_println(t *testing.T) {
	defer Wait()

	SetAlsoToStdout(true)

	logger = NewLogger("test", 1, 1, 1, true)

	logger.Println("1")

	logger.Println("1", "2")

	d := 4
	logger.Println("1", 3)

	logger.Println(3, "2", d)
}

func Test_2logger_by_suffix(t *testing.T) {
	defer Wait()

	for i := 0; i < 100; i++ {
		l1 := GetLoggerBySuffix("a.log", "./logs/", 1, 1, 1, true)
		l2 := GetLoggerBySuffix("b.log", "./logs/", 1, 1, 1, false)

		t.Log("print 1")

		l2.Print("aaaa")

		t.Log("flush 1")

		l2.Flush()

		t.Log("print 2")

		l1.Print("bbb")

		t.Log("flush 2")

		l1.Flush()
		l2.Flush()
	}

}
