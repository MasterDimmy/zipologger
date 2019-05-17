package zipologger

/*
	Набор функций для ротационного журналирования
*/

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/MasterDimmy/zilorot"
)

type logger_message struct {
	msg         string
	to_file     bool
	to_email    bool
	email_theme string
}

type Logger struct {
	log *log.Logger
	wg  sync.WaitGroup
	ch  chan *logger_message
}

var inited_loggers []*Logger

//create logger
func NewLogger(filename string, log_max_size_in_mb int, max_backups int, max_age_in_days int) *Logger {
	p := filepath.Dir(filename)
	os.MkdirAll(p, 0666)
	log := Logger{
		log: newLogger(filename, log_max_size_in_mb, max_backups, max_age_in_days),
	}
	log.init()
	inited_loggers = append(inited_loggers, &log)
	return &log
}

func Wait() {
	for _, v := range inited_loggers {
		v.Wait()
	}
}

var wait_wg sync.Mutex
//waiting till all will be writed
func (l *Logger) Wait() {
	wait_wg.Lock()
	defer wait_wg.Unlock()
	l.wg.Wait()
}

//order and log to the file
func (l *Logger) init() {
	l.ch = make(chan *logger_message, 10)
	go func() {
		for elem := range l.ch {
			func() {
				defer l.wg.Done()

				str := elem.msg

				if !strings.HasSuffix(str, "\n") {
					str += "\n"
				}
				if l.log != nil {
					l.log.Printf(str)
				} else {
					panic("cant printf to log file")
				}

			}()
		}
	}()
}

func (l *Logger) print(format string) string {
	l.wg.Add(1)
	l.ch <- &logger_message{
		msg: format,
	}
	return format
}

func (l *Logger) Print(format string) string {
	return l.print(format)
}

func (l *Logger) printf(format string, w1 interface{}, w2 ...interface{}) string {
	l.wg.Add(1)
	var w3 []interface{}
	w3 = append(w3, w1)
	if len(w2) > 0 {
		for _, v := range w2 {
			w3 = append(w3, v)
		}
	}
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	msg := fmt.Sprintf(format, w3...)
	l.ch <- &logger_message{
		msg: msg,
	}
	return msg
}

func (l *Logger) Printf(format string, w1 interface{}, w2 ...interface{}) string {
	return l.printf(format, w1, w2...)
}

var panic_mutex sync.Mutex

//intercept panics and save it to file
func HandlePanic(err_log *Logger, e interface{}) string {
	panic_mutex.Lock()
	defer panic_mutex.Unlock()
	str := savePanicToFile(fmt.Sprintf("%s", e))
	fmt.Printf("PANIC: %s\n", str)
	if err_log != nil {
		err_log.Printf("PANIC: %s\n", e)
	}
	return str
}

// Current stack dump
func Stack() string {
	b := make([]byte, 1<<16)
	written := runtime.Stack(b, true)
	return string(b[:written])
}

func savePanicToFile(pdesc string) string {
	dts := time.Now().Format("2006-01-02(15.04.05.000)")
	st, _ := filepath.Abs(os.Args[0])
	stname := filepath.Base(os.Args[0])
	os.Mkdir("logs", 0777)
	fn := filepath.Join(filepath.Dir(st), "logs/panic_"+stname+"_"+dts+".log")
	f, e := os.Create(fn)
	if e == nil {
		defer f.Close()
		_, file, line, _ := runtime.Caller(1)
		str := fmt.Sprintf("Panic in [%s:%d] :\n", file, line) + pdesc + "\nSTACK:\n" + Stack()
		f.WriteString(str)
		return str
	}
	return ""
}

func newLogger(name string, log_max_size_in_mb int, max_backups int, max_age_in_days int) *log.Logger {
	e, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		fmt.Printf("error opening file: %v", err)
		os.Exit(1)
	}
	logg := log.New(e, "", log.Ldate|log.Ltime)

	if logg != nil {
		logg.SetOutput(&zilorot.Logger{
			Filename:   name,
			MaxSize:    log_max_size_in_mb, // megabytes
			MaxBackups: max_backups,
			MaxAge:     max_age_in_days, //days
		})
	}

	return logg
}
