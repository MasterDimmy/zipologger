package zipologger

/*
	Набор функций для ротационного журналирования
*/

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	log          *log.Logger
	zlog         *zilorot.Logger
	wait_started int32
	m            sync.Mutex

	//файл будет создан при первой записи, чтобы не делать пустышки
	filename           string
	log_max_size_in_mb int
	max_backups        int
	max_age_in_days    int
	write_fileline     bool

	alsoToStdout bool

	log_tasks sync.WaitGroup //сколько сообщений в очереди?

	limited_print *lru.Cache //printid - unixitime
}

var tolog_ch = make(chan *logger_message, 1000)

//order and log to the file
func init() {
	go func() {
		defer HandlePanic()

		//remove parallel write due to order breaking
		for elem := range tolog_ch {
			if elem.log.log == nil { //create file to be written
				elem.log.log, elem.log.zlog = newLogger(elem.log.filename, elem.log.log_max_size_in_mb, elem.log.max_backups, elem.log.max_age_in_days)
			}

			str := elem.msg

			if !strings.HasSuffix(str, "\n") {
				str += "\n"
			}

			elem.log.log.Print(str)

			elem.log.log_tasks.Done()
		}
	}()
}

//you can set Logger to this to write nowhere
var EmptyLogger = func() *Logger {
	nowhere := &log.Logger{}
	nowhere.SetOutput(ioutil.Discard)
	return &Logger{
		log:      nowhere,
		filename: "",
	}
}()

var alsoToStdout bool

func SetAlsoToStdout(b bool) {
	alsoToStdout = b
}

var closing_log_files sync.WaitGroup
var inited_loggers, _ = lru.NewWithEvict(50, func(key interface{}, value interface{}) {
	log := value.(*Logger)

	closing_log_files.Add(1)
	go func() {
		defer closing_log_files.Done()
		if log.zlog != nil {
			log.zlog.Close()
		}
	}()
})
var newLogger_mutex sync.Mutex

func NewLogger(filename string, log_max_size_in_mb int, max_backups int, max_age_in_days int, write_fileline bool) *Logger {
	newLogger_mutex.Lock()
	defer newLogger_mutex.Unlock()

	logger, ok := inited_loggers.Get(filename)
	if ok {
		inited_loggers.Add(filename, logger)
		return logger.(*Logger)
	}

	p := filepath.Dir(filename)
	os.MkdirAll(p, 0644)
	l, _ := lru.New(1000)
	log := &Logger{
		filename:           filename,
		log_max_size_in_mb: log_max_size_in_mb,
		max_backups:        max_backups,
		max_age_in_days:    max_age_in_days,
		write_fileline:     write_fileline,
		limited_print:      l,
	}

	inited_loggers.Add(filename, log)
	return log
}

//делает flush
var w_mutex sync.Mutex

func Wait() {
	w_mutex.Lock()
	defer w_mutex.Unlock()
	for _, w := range inited_loggers.Keys() {
		log, ok := inited_loggers.Get(w)
		if ok {
			logger := log.(*Logger)
			logger.Wait()
		}
	}
	closing_log_files.Wait()
}

func (l *Logger) Writer() io.Writer {
	if l.log != nil {
		return l.log.Writer()
	} else {
		return nil
	}
}

func (l *Logger) Flush() {
	l.Wait()
}

//waiting till all will be written
func (l *Logger) Wait() {
	l.m.Lock()
	defer l.m.Unlock()
	atomic.StoreInt32(&l.wait_started, 1)
	l.log_tasks.Wait()
	atomic.StoreInt32(&l.wait_started, 0)
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

	if len(ret) > 5 {
		ret += "\n"
	}

	return ret
}

func (l *Logger) print(msg string) string {
	if atomic.LoadInt32(&l.wait_started) > 0 { //forbid add if wait called!
		return msg
	}

	if l.write_fileline {
		msg = formatCaller(GetStartCallerDepth()) + msg
	}

	l.log_tasks.Add(1)

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
	return l.print(format)
}

func (l *Logger) printf(format string, w1 interface{}, w2 ...interface{}) string {
	var w3 []interface{}
	w3 = append(w3, w1)
	if len(w2) > 0 {
		for _, v := range w2 {
			w3 = append(w3, v)
		}
	}

	return l.print(fmt.Sprintf(format, w3...))
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

func newLogger(name string, log_max_size_in_mb int, max_backups int, max_age_in_days int) (*log.Logger, *zilorot.Logger) {
	e, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("error opening file: %v", err))
		os.Exit(1)
	}
	logg := log.New(e, "", log.Ldate|log.Ltime)

	var output *zilorot.Logger

	if logg != nil {
		output = &zilorot.Logger{
			Filename:   name,
			MaxSize:    log_max_size_in_mb, // megabytes
			MaxBackups: max_backups,
			MaxAge:     max_age_in_days, //days
		}
		logg.SetOutput(output)
	}

	return logg, output
}

var ErrorCatcher *errorcatcher.System

func HandlePanic() {
	if e := recover(); e != nil {
		p := fmt.Sprintf("%v", e)
		fmt.Printf(p)
		sp := savePanicToFile(p)
		if ErrorCatcher != nil {
			ErrorCatcher.Send(sp)
			time.Sleep(100 * time.Millisecond)
			ErrorCatcher.Wait()
		}
	}
}

//автоматически создает и возвращает логгер на файл с заданным суффиксом
func GetLoggerBySuffix(suffix string, name string, log_max_size_in_mb int, max_backups int, max_age_in_days int, write_source bool) *Logger {
	return NewLogger(name+suffix, log_max_size_in_mb, max_backups, max_age_in_days, write_source)
}
