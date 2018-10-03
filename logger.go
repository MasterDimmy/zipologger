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
	log        *log.Logger
	wg         sync.WaitGroup
	ch         chan *logger_message
	debug      bool
	email_func func(string, string)
}

//create logger
func NewLogger(filename string, log_max_size_in_mb int, max_backups int, max_age_in_days int, debug bool, email_func func(string, string)) *Logger {
	log := Logger{
		log:        newLogger(filename, log_max_size_in_mb, max_backups, max_age_in_days),
		debug:      debug,
		email_func: email_func,
	}
	log.init()
	return &log
}

//waiting till all will be writed
func (l *Logger) Wait() {
	l.wg.Wait()
}

//упорядочиваем журнал по времени прибытия
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
				if l.debug {
					fmt.Printf(str)
				}
				if elem.to_file && l.log != nil {
					l.log.Printf(str)
				}
				if elem.to_email && l.email_func != nil {
					l.email_func(elem.email_theme, str)
				}
			}()
		}
	}()
}

//print group: fmt.printf to stdout

func (l *Logger) print(format string, to_file bool, to_email bool, email_theme string) {
	l.wg.Add(1)
	l.ch <- &logger_message{
		msg:         format,
		to_file:     to_file,
		to_email:    to_email,
		email_theme: email_theme,
	}
}

//always print to file, never email
func (l *Logger) Print(format string) {
	l.print(format, true, false, "")
}

//If debug is on - to stdout & to file, never email
func (l *Logger) Debugp(format string) {
	if l.debug {
		l.print(format, true, false, "")
	}
}

//Log error to file & email it, if debug - print to screen
func (l *Logger) Errp(format string) {
	l.print(format, true, true, "ERROR")
}

//Print to stdout, if debug, then to file, no email
func (l *Logger) Adminp(format string) {
	l.print(format, true, true, "ADMIN")
}

func (l *Logger) printf(format string, to_file bool, to_email bool, email_theme string, w1 interface{}, w2 ...interface{}) {
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
	l.ch <- &logger_message{
		msg:      fmt.Sprintf(format, w3...),
		to_email: to_email,
		to_file:  to_file,
	}
}

//Always print to file, never email
func (l *Logger) Printf(format string, w1 interface{}, w2 ...interface{}) {
	l.printf(format, true, false, "", w1, w2...)
}

//If debug is on - to stdout & to file, never email
func (l *Logger) Debugpf(format string, w1 interface{}, w2 ...interface{}) {
	if l.debug {
		l.printf(format, l.debug, false, "", w1, w2...)
	}
}

//Log error to file & email it, if debug - print to screen
func (l *Logger) Errpf(format string, w1 interface{}, w2 ...interface{}) {
	l.printf(format, true, true, "ERROR", w1, w2...)
}

//Print to stdout, if debug, then to file, no email
func (l *Logger) Adminpf(format string, w1 interface{}, w2 ...interface{}) {
	l.printf(format, true, true, "ADMIN", w1, w2...)
}

var panic_mutex sync.Mutex

//intercept panics and save it to file
func HandlePanic(err_log *Logger) {
	if e := recover(); e != nil {
		panic_mutex.Lock()
		defer panic_mutex.Unlock()
		if err_log != nil {
			err_log.Errpf("PANIC: %s", e)
		}
		savePanicToFile(fmt.Sprintf("%s", e))
		//dumpMem(nil, nil)
		panic(e)
	}
}

// Current stack dump
func Stack() string {
	b := make([]byte, 1<<16)
	written := runtime.Stack(b, true)
	return string(b[:written])
}

func savePanicToFile(pdesc string) {
	dts := time.Now().Format("2006-01-02(15.04.05.000)")
	st, _ := filepath.Abs(os.Args[0])
	os.Mkdir("logs", 0777)
	fn := filepath.Join(filepath.Dir(st), "logs/panic_"+os.Args[0]+"_"+dts+".log")
	f, e := os.Create(fn)
	if e == nil {
		defer f.Close()
		_, file, line, _ := runtime.Caller(1)
		f.WriteString(fmt.Sprintf("Panic in [%s:%d] :\n", file, line) + pdesc + "\nSTACK:\n")
		f.WriteString(Stack())
	}
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
