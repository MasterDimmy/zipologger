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
	"github.com/MasterDimmy/golang-lruexpire"
	"github.com/MasterDimmy/zilorot"
)

type logger_message struct {
	msg string
	log *Logger
}

type Logger struct {
	log *log.Logger

	//файл будет создан при первой записи, чтобы не делать пустышки
	filename           string
	log_max_size_in_mb int
	max_backups        int
	max_age_in_days    int
	write_fileline     bool

	alsoToStdout bool

	wg             sync.WaitGroup //current_writes
	stopwait_mutex sync.Mutex

	limited_print *lru.Cache //printid - unixitime
}

var alsoToStdout bool
var purge_time = time.Second * 10 //логгер через указанное время будет удален

func SetAlsoToStdout(b bool) {
	alsoToStdout = b
}

var inited_loggers, _ = lru.NewWithExpire(100, purge_time)
var newLogger_mutex sync.Mutex

var init_logger_once sync.Once
var tolog_ch = make(chan *logger_message, 1000)

func NewLogger(filename string, log_max_size_in_mb int, max_backups int, max_age_in_days int, write_fileline bool) *Logger {
	newLogger_mutex.Lock()
	defer newLogger_mutex.Unlock()

	logger, ok := inited_loggers.Get(filename)
	if ok {
		inited_loggers.Add(filename, logger)
		return logger.(*Logger)
	}

	p := filepath.Dir(filename)
	os.MkdirAll(p, 0666)
	l, _ := lru.New(1000)
	log := &Logger{
		filename:           filename,
		log_max_size_in_mb: log_max_size_in_mb,
		max_backups:        max_backups,
		max_age_in_days:    max_age_in_days,
		write_fileline:     write_fileline,
		limited_print:      l,
	}

	go init_logger_once.Do(init_lib)

	inited_loggers.Add(filename, log)
	return log
}

//делает flush
var w_mutex sync.Mutex

func Wait() {
	w_mutex.Lock()
	defer w_mutex.Unlock()
	for w := range inited_loggers.Keys() {
		log, ok := inited_loggers.Get(w)
		if ok {
			logger := log.(*Logger)
			logger.wait()
		}
	}
}

func (l *Logger) Flush() {
	l.wait()
}

//waiting till all will be written
func (l *Logger) wait() {
	l.stopwait_mutex.Lock()
	defer l.stopwait_mutex.Unlock()
	l.wg.Add(1)
	l.wg.Done()
	l.wg.Wait()
}

//order and log to the file
func init_lib() {
	m := sync.Mutex{}
	for el := range tolog_ch {
		el.log.wg.Add(1)
		go func(elem *logger_message) {
			defer elem.log.wg.Done()

			m.Lock()
			if elem.log.log == nil { //create file to be written
				elem.log.log = newLogger(elem.log.filename, elem.log.log_max_size_in_mb, elem.log.max_backups, elem.log.max_age_in_days)
			}
			m.Unlock()

			str := elem.msg

			if !strings.HasSuffix(str, "\n") {
				str += "\n"
			}

			elem.log.log.Print(str)
		}(el)
	}
}

func (l *Logger) SetAlsoToStdout(b bool) {
	l.alsoToStdout = b
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

	tolog_ch <- &logger_message{
		msg: msg, //1 call
		log: l,
	}

	if alsoToStdout || l.alsoToStdout {
		fmt.Println(msg)
	}

	return msg
}

func (l *Logger) Print(format string) string {
	return l.print(format) //2 cal l
}

func (l *Logger) printf(format string, w1 interface{}, w2 ...interface{}) string {
	l.stopwait_mutex.Lock()
	defer l.stopwait_mutex.Unlock()

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

	tolog_ch <- &logger_message{
		msg: msg,
		log: l,
	}

	if alsoToStdout || l.alsoToStdout {
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

func (l *Logger) Println(w ...interface{}) string {
	switch len(w) {
	case 0:
		return ""
	case 1:
		return l.printf("%v\n", w[0])
	case 2:
		return l.printf("%v %v\n", w[0], w[1])
	default:
		tail := ""
		for _ = range w {
			tail += "%v "
		}
		if len(tail) > 0 {
			tail = tail[:len(tail)-1]
		}
		return l.printf(tail+"\n", w[0], w[1:]...)
	}
}

func (l *Logger) Fatalf(format string, w1 interface{}, w2 ...interface{}) {
	ret := l.printf(format, w1, w2...)
	l.Flush()
	panic(ret)
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
	fn := filepath.Join(filepath.Dir(st), "logs/panic_"+filepath.Base(os.Args[0])+time.Now().Format("_2006-Jan-02_15")+".log")
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
		savePanicToFile(p)
		if ErrorCatcher != nil {
			ErrorCatcher.Send(p)
			time.Sleep(100 * time.Millisecond)
			ErrorCatcher.Wait()
		}
	}
}

//автоматически создает и возвращает логгер на файл с заданным суффиксом
var get_logger_by_suffix_mutex sync.Mutex
var loggers_by_suffix, _ = lru.NewWithExpire(100, purge_time) //если не использовался - будет удален!!!

func GetLoggerBySuffix(suffix string, name string, log_max_size_in_mb int, max_backups int, max_age_in_days int, write_source bool) *Logger {
	defer HandlePanic()

	get_logger_by_suffix_mutex.Lock()
	defer get_logger_by_suffix_mutex.Unlock()

	log, ok := loggers_by_suffix.Get(suffix)
	if ok {
		loggers_by_suffix.Add(suffix, log) //update expire cache
		return log.(*Logger)
	}
	new_log := NewLogger(name+suffix, log_max_size_in_mb, max_backups, max_age_in_days, write_source)
	loggers_by_suffix.Add(suffix, new_log)
	return new_log
}
