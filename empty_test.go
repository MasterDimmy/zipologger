package zipologger

import (
	"os"
	"strings"
	"sync"
	"testing"
)

func Test_EmptyLogger(t *testing.T) {
	defer Wait()

	SetAlsoToStdout(false)

	os.Chdir("C:\\gopath\\src\\github.com\\MasterDimmy\\zipologger\\")

	err := os.Remove("./logs/a.log")
	if err != nil && !strings.Contains(err.Error(), "The system cannot find the file") {
		t.Fatal(err.Error())
	}
	err = os.Remove("./logs/b.log")
	if err != nil && !strings.Contains(err.Error(), "The system cannot find the file") {
		t.Fatal(err.Error())
	}

	for i := 0; i < 100; i++ {
		l1 := GetLoggerBySuffix("a.log", "./logs/", 1, 1, 1, false)
		l2 := GetLoggerBySuffix("b.log", "./logs/", 1, 1, 1, false)

		//t.Log("print 1")

		st := l2.Print("aaaa")
		if st != "aaaa" {
			t.Fatal()
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

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		l1 := GetLoggerBySuffix("a.log", "./logs/", 1, 1, 1, false)
		l2 := GetLoggerBySuffix("b.log", "./logs/", 1, 1, 1, false)

		wg.Add(1)
		go func() {
			l1.Printf("[%d]", i)
			l2.Printf("[%d]", i)
			l1.Printf("[%d]", i)
			wg.Add(1)
			go func() {
				l2.Println("zzzz")
				wg.Done()
			}()
			wg.Done()
		}()
	}

	empty := EMPTY_LOGGER
	empty.Print("te")
	empty.Flush()
	empty.Wait()

	wg.Wait()

	Wait()

	tf := func(fname string) {
		a1, err := os.ReadFile(fname)
		if err != nil {
			t.Fatal(err.Error())
		}
		astr := strings.TrimSpace(string(a1))
		al := strings.Split(astr, "\n")
		if len(al) != 300 {
			t.Fatalf("%s not 300 lines: %d\n", fname, len(al))
		}
	}

	tf("./logs/a.log")
	tf("./logs/b.log")
}
