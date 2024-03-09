package zipologger

import (
	"fmt"
	"testing"
	"time"
)

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

	t.Log("wait 1")
	Wait()
	t.Log("wait 2")
	Wait()
}

func Test_2logger_by_suffix(t *testing.T) {
	defer Wait()

	SetAlsoToStdout(false)

	for i := 0; i < 10000; i++ {
		l1 := GetLoggerBySuffix("a.log", "./logs/", 1, 1, 1, false)
		l2 := GetLoggerBySuffix("b.log", "./logs/", 1, 1, 1, false)

		//t.Log("print 1")

		st := l2.Printf("%d aaaa - %s", i, time.Now().String())
		if i%100 == 0 {
			t.Log(st)
		}

		//t.Log("flush 1")

		l2.Flush()

		//t.Log("print 2")

		l1.Print("bbb")

		//t.Log("flush 2")

		l1.Flush()
		l2.Flush()
		l1.Wait()
		l2.Wait()
		l1.Wait()
		l1.Wait()
	}

}

func Test_CloseFiles(t *testing.T) {
	defer Wait()

	sw := GetLoggerBySuffix(fmt.Sprintf("123123.log"), "./logs/", 1, 1, 1, false)
	sw.Print("123123123")

	Wait()

	SetAlsoToStdout(false)

	for i := 0; i < 100; i++ {
		l1 := GetLoggerBySuffix(fmt.Sprintf("a_%d.log", i), "./logs/", 1, 1, 1, false)
		l1.Printf("%d a", i)
	}

	Wait()
	Wait()
}
