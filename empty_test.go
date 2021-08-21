package zipologger

import (
	"os"
	"strings"
	"sync"
	"testing"
)

func init() {
	os.Chdir("C:\\gopath\\src\\github.com\\MasterDimmy\\zipologger\\")
}

func remove_logs() {
	err := os.Remove("./logs/a.log")
	if err != nil && !strings.Contains(err.Error(), "The system cannot find the file") {
		panic(err.Error())
	}
	err = os.Remove("./logs/b.log")
	if err != nil && !strings.Contains(err.Error(), "The system cannot find the file") {
		panic(err.Error())
	}
}

func Test_wait(t *testing.T) {
	SetAlsoToStdout(false)

	remove_logs()

	l1 := GetLoggerBySuffix("a.log", "./logs/", 1, 1, 1, true)
	l2 := GetLoggerBySuffix("b.log", "./logs/", 1, 1, 1, true)

	l1.Wait()
	l1.Wait()
	l2.Wait()
	l1.Wait()

	for i := 0; i < 100; i++ {
		l2.Print("aaaa")
		l1.Print("bbb")
	}

	Wait()

	tf := func(fname string, cnt int) {
		a1, err := os.ReadFile(fname)
		if err != nil {
			t.Fatal(err.Error())
		}
		astr := strings.TrimSpace(string(a1))
		al := strings.Split(astr, "\n")
		if len(al) != cnt {
			t.Fatalf("%s not %d lines: %d\n", fname, cnt, len(al))
		}
	}

	tf("./logs/a.log", 100)
	tf("./logs/b.log", 100)

	for i := 0; i < 100; i++ {
		//t.Log("print 1")

		l2.Print("aaaa")
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
	for ii := 0; ii < 100; ii++ {
		go func(i int) {
			wg.Add(1)
			go func(j int) {
				l1.Printf("[%d]", j)
				l2.Printf("[%d]", j)
				l1.Printf("[%d]", j)
				wg.Add(1)
				go func() {
					l2.Println("zzzz")
					wg.Done()
				}()
				wg.Done()
			}(i)
		}(ii)

	}

	empty := EMPTY_LOGGER
	empty.Print("te")
	empty.Flush()
	empty.Wait()

	wg.Wait()

	for i := 0; i < 100; i++ {
		l1.Printf("zxswq [%d]", i)
	}

	for i := 0; i < 100; i++ {
		l2.Printf("tgbf [%d]", i)
	}

	Wait()

	tf("./logs/a.log", 500)
	tf("./logs/b.log", 500)
}
