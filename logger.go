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

	"github.com/MasterDimmy/errorcatcher"
	"github.com/MasterDimmy/zilorot"
	"github.com/hashicorp/golang-lru"
)

type logger_message struct {
	msg         string
	to_file     bool
	to_email    bool
	email_theme string
}

type Logger struct {
	log *log.Logger

	//файл будет создан при первой записи, чтобы не делать пустышки
	filename           string
	log_max_size_in_mb int
	max_backups        int
	max_age_in_days    int
	write_fileline     bool

	wg             sync.WaitGroup
	stopwait_mutex sync.Mutex

	ch chan *logger_message

	limited_print *lru.Cache //printid - unixitime
}

var inited_loggers []*Logger

var alsoToStdout bool

func SetAlsoToStdout(b bool) {
	alsoToStdout = b
}

//create logger
func NewLogger(filename string, log_max_size_in_mb int, max_backups int, max_age_in_days int, write_fileline bool) *Logger {
	p := filepath.Dir(filename)
	os.MkdirAll(p, 0666)
	l, _ := lru.New(1000)
	log := Logger{
		filename:           filename,
		log_max_size_in_mb: log_max_size_in_mb,
		max_backups:        max_backups,
		max_age_in_days:    max_age_in_days,
		write_fileline:     write_fileline,
		limited_print:      l,
	}
	log.init()
	inited_loggers = append(inited_loggers, &log)
	return &log
}

//делает flush
func Wait() {
	for _, v := range inited_loggers {
		v.wait()
	}
}

func (l *Logger) Flush() {
	l.wait()
}

//waiting till all will be writed
func (l *Logger) wait() {
	l.stopwait_mutex.Lock()
	defer l.stopwait_mutex.Unlock()
	l.wg.Wait()
}

//order and log to the file
func (l *Logger) init() {
	l.ch = make(chan *logger_message, 1000)
	for i := 1; i < 2; i++ { //в 2 треда на журнал
		go func() {
			for elem := range l.ch {
				func() {
					defer l.wg.Done()

					str := elem.msg

					if !strings.HasSuffix(str, "\n") {
						str += "\n"
					}
					if l.log == nil {
						l.log = newLogger(l.filename, l.log_max_size_in_mb, l.max_backups, l.max_age_in_days)
					}

					if l.log != nil {
						l.log.Print(str)
					} else {
						panic("cant printf to log file")
					}
				}()
			}
		}()
	}
}

var start_caller_depth int
var max_caller_depth = 7
var additional_caller_depth_m sync.Mutex

func SetStartCallerDepth(a int) {
	additional_caller_depth_m.Lock()
	defer additional_caller_depth_m.Unlock()
	start_caller_depth = a
}

func GetStartCallerDepth() int {
	additional_caller_depth_m.Lock()
	defer additional_caller_depth_m.Unlock()
	return start_caller_depth
}

func SetMaxCallerDepth(a int) {
	additional_caller_depth_m.Lock()
	defer additional_caller_depth_m.Unlock()
	max_caller_depth = a
}

func GetMaxCallerDepth() int {
	additional_caller_depth_m.Lock()
	defer additional_caller_depth_m.Unlock()
	return max_caller_depth
}

func formatCaller(add int) string {
	ret := ""
	previous := ""
	for i := GetMaxCallerDepth() + add; i >= 3+add; i-- {
		_, file, line, ok := runtime.Caller(i) //0 call
		if !ok {
			file = "???"
			line = 0
		} else {
			if !strings.HasSuffix(file, "src/testing/testing.go") {
				if !strings.HasSuffix(file, "runtime/asm_amd64.s") && !strings.HasSuffix(file, "runtime/proc.go") {
					t := strings.LastIndex(file, "/")
					if t > 0 {
						file = file[t+1:]
					}
					if len(ret) > 0 {
						ret = ret + "=>"
					}
					if previous == file {
						ret = ret + fmt.Sprintf(":%d", line)
					} else {
						ret = ret + fmt.Sprintf("%s", fmt.Sprintf("%s:%d", file, line))
					}
					previous = file
				}
			}
		}
	}

	//глубже ничего нет
	if len(ret) < 3 && add > 0 {
		ret = formatCaller(0)
	} else {
		ret = ret + ": "
	}

	return ret
}

func (l *Logger) print(format string) string {
	l.stopwait_mutex.Lock()
	defer l.stopwait_mutex.Unlock()

	msg := format
	if l.write_fileline {
		msg = formatCaller(GetStartCallerDepth()) + msg
	}

	l.wg.Add(1)
	l.ch <- &logger_message{
		msg: msg, //1 call
	}

	if alsoToStdout {
		fmt.Println(msg)
	}

	return format
}

func (l *Logger) Print(format string) string {
	return l.print(format) //2 call
}

func (l *Logger) printf(format string, w1 interface{}, w2 ...interface{}) string {
	l.stopwait_mutex.Lock()
	defer l.stopwait_mutex.Unlock()

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

	if l.write_fileline {
		msg = formatCaller(GetStartCallerDepth()) + msg
	}

	l.ch <- &logger_message{
		msg: msg,
	}

	if alsoToStdout {
		fmt.Println(msg)
	}

	return msg
}

//выводит на печать первый вызов, затем не чаще, чем duration для данной строки printid
func (l *Logger) LimitedPrintf(printid string, duration time.Duration, format string, w1 interface{}, w2 ...interface{}) {
	old, ok := l.limited_print.Get(printid)
	if ok {
		old_v := old.(time.Time)
		if time.Since(old_v) < duration {
			return
		}
	}
	l.limited_print.Add(printid, time.Now())
	l.printf(format, w1, w2...)
}

func (l *Logger) Printf(format string, w1 interface{}, w2 ...interface{}) string {
	return l.printf(format, w1, w2...)
}

var panic_mutex sync.Mutex

//intercept panics and save it to file
func HandlePanicLog(err_log *Logger, e interface{}) string {
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
	st, _ := filepath.Abs(os.Args[0])
	os.Mkdir("logs", 0777)
	fn := filepath.Join(filepath.Dir(st), "logs/panic_"+filepath.Base(os.Args[0])+".log")
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

var ErrorCatcher *errorcatcher.System

func HandlePanic() {
	if e := recover(); e != nil {
		p := fmt.Sprintf("%v", e)
		fmt.Printf(p)
		if ErrorCatcher != nil {
			ErrorCatcher.Send(p)
		}
		savePanicToFile(p)
	}
}

//автоматически создает и возвращает логгер на файл с заданным суффиксом
var get_logger_by_suffix_mutex sync.Mutex
var loggers_by_suffix = make(map[string]*Logger)

func GetLoggerBySuffix(suffix string, name string, log_max_size_in_mb int, max_backups int, max_age_in_days int, write_source bool) *Logger {
	defer HandlePanic()
	get_logger_by_suffix_mutex.Lock()
	defer get_logger_by_suffix_mutex.Unlock()

	log, ok := loggers_by_suffix[suffix]
	if ok {
		return log
	}
	new_log := NewLogger(name+suffix, log_max_size_in_mb, max_backups, max_age_in_days, write_source)
	loggers_by_suffix[suffix] = new_log
	return new_log
}
