package zipologger

import (
	"encoding/base64"
	"fmt"
	"io"
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
	"github.com/MasterDimmy/zipologger/enc"
)

type loggerMessage struct {
	msg string
	log *Logger
}

type Logger struct {
	log            *log.Logger
	zlog           *zilorot.Logger
	waitStarted    int32
	m              sync.Mutex
	em             sync.Mutex
	encryptionKey  *enc.KeyEncrypt
	filename       string
	logMaxSizeInMB int
	maxBackups     int
	maxAgeInDays   int
	alsoToStdout   bool
	logTasks       sync.WaitGroup
	limitedPrint   *lru.Cache //printid - unixitime
	logDateTime    bool
	logSourcePath  bool
}

var (
	tologCh        = make(chan *loggerMessage, 1000)
	alsoToStdout   bool
	initedLoggers  *lru.Cache
	newLoggerMutex sync.Mutex
	panicMutex     sync.Mutex
	wMutex         sync.Mutex
)

func init() {
	initedLoggers, _ = lru.NewWithEvict(1000, func(key interface{}, value interface{}) {
		log := value.(*Logger)
		if log != nil && log.zlog != nil {
			log.Flush()
			log.zlog.Close()
		}
	})

	go func() {
		defer HandlePanic()

		for {
			select {
			case elem := <-tologCh:
				elem.log.em.Lock()
				if elem.log.log == nil {
					elem.log.log, elem.log.zlog = newLogger(elem.log.filename, elem.log.logMaxSizeInMB, elem.log.maxBackups, elem.log.maxAgeInDays)
				}
				elem.log.em.Unlock()

				str := elem.msg

				for strings.HasSuffix(str, "\n") {
					str = strings.TrimSuffix(str, "\n")
				}

				globalEncryptor.m.Lock()
				elem.log.em.Lock()
				enckey := elem.log.encryptionKey
				if enckey == nil {
					enckey = globalEncryptor.key
				}
				elem.log.em.Unlock()
				globalEncryptor.m.Unlock()

				if enckey != nil {
					ret, err := enckey.EncryptString(str)
					if err == nil {
						str = base64.RawStdEncoding.EncodeToString(ret)
					}
				}

				str = str + "\n"
				elem.log.log.Print(str)
				elem.log.logTasks.Done()
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
}

// EmptyLogger is a logger that writes nowhere
var EmptyLogger = func() *Logger {
	nowhere := &log.Logger{}
	nowhere.SetOutput(io.Discard)
	return &Logger{
		log:      nowhere,
		filename: "",
	}
}()

func (l *Logger) WriteSourcePath(b bool) *Logger {
	l.m.Lock()
	defer l.m.Unlock()
	l.logSourcePath = b
	return l
}

func (l *Logger) WriteDateTime(b bool) *Logger {
	l.m.Lock()
	defer l.m.Unlock()
	l.logDateTime = b
	return l
}

func (l *Logger) SetAlsoToStdout(b bool) *Logger {
	l.m.Lock()
	defer l.m.Unlock()
	l.alsoToStdout = b
	return l
}

func SetAlsoToStdout(b bool) {
	alsoToStdout = b
}

func NewLogger(filename string, logMaxSizeInMB int, maxBackups int, maxAgeInDays int, writeFileline bool) *Logger {
	newLoggerMutex.Lock()
	defer newLoggerMutex.Unlock()

	logger, ok := initedLoggers.Get(filename)
	if ok {
		initedLoggers.Add(filename, logger)
		return logger.(*Logger)
	}

	p := filepath.Dir(filename)
	os.MkdirAll(p, 0755)
	l, _ := lru.New(1000)
	log := &Logger{
		filename:       filename,
		logMaxSizeInMB: logMaxSizeInMB,
		maxBackups:     maxBackups,
		maxAgeInDays:   maxAgeInDays,
		logSourcePath:  writeFileline,
		limitedPrint:   l,
		logDateTime:    true,
	}

	initedLoggers.Add(filename, log)
	return log
}

func Wait() {
	wMutex.Lock()
	defer wMutex.Unlock()
	for _, w := range initedLoggers.Keys() {
		log, ok := initedLoggers.Get(w)
		if ok {
			logger := log.(*Logger)
			logger.Wait()
		}
	}
}

func (l *Logger) Writer() io.Writer {
	if l.log != nil {
		return l.log.Writer()
	}
	return nil
}

func (l *Logger) Flush() {
	l.Wait()
}

func (l *Logger) Wait() {
	l.m.Lock()
	defer l.m.Unlock()
	atomic.StoreInt32(&l.waitStarted, 1)
	l.logTasks.Wait()
	atomic.StoreInt32(&l.waitStarted, 0)
}

var startCallerDepth int
var maxCallerDepth = 7
var additionalCallerDepthM sync.Mutex

func SetStartCallerDepth(a int) {
	additionalCallerDepthM.Lock()
	defer additionalCallerDepthM.Unlock()
	startCallerDepth = a
}

func GetStartCallerDepth() int {
	additionalCallerDepthM.Lock()
	defer additionalCallerDepthM.Unlock()
	return startCallerDepth
}

func SetMaxCallerDepth(a int) {
	additionalCallerDepthM.Lock()
	defer additionalCallerDepthM.Unlock()
	maxCallerDepth = a
}

func GetMaxCallerDepth() int {
	additionalCallerDepthM.Lock()
	defer additionalCallerDepthM.Unlock()
	return maxCallerDepth
}

func formatCaller(add int) string {
	ret := ""
	previous := ""
	for i := GetMaxCallerDepth() + add; i >= 3+add; i-- {
		_, file, line, ok := runtime.Caller(i)
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
	if atomic.LoadInt32(&l.waitStarted) > 0 {
		return msg
	}

	if l.logSourcePath {
		msg = formatCaller(GetStartCallerDepth()) + msg
	}

	if l.logDateTime {
		msg = time.Now().Format("2006/01/02 15:04:05 ") + msg
	}

	l.logTasks.Add(1)

	tologCh <- &loggerMessage{
		msg: msg,
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
	w3 := append([]interface{}{w1}, w2...)
	return l.print(fmt.Sprintf(format, w3...))
}

func (l *Logger) LimitedPrintf(printid string, duration time.Duration, format string, w1 interface{}, w2 ...interface{}) {
	old, ok := l.limitedPrint.Get(printid)
	if ok {
		oldV := old.(time.Time)
		if time.Since(oldV) < duration {
			return
		}
	}
	l.limitedPrint.Add(printid, time.Now())
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
		tail := strings.Repeat("%v ", len(w))
		return l.printf(tail[:len(tail)-1]+"\n", w[0], w[1:]...)
	}
}

func (l *Logger) Fatalf(format string, w1 interface{}, w2 ...interface{}) {
	ret := l.printf(format, w1, w2...)
	l.Flush()
	panic(ret)
}

func HandlePanicLog(errLog *Logger, e interface{}) string {
	panicMutex.Lock()
	defer panicMutex.Unlock()
	str := savePanicToFile(fmt.Sprintf("%s", e))
	fmt.Printf("PANIC: %s\n", str)
	if errLog != nil {
		errLog.Printf("PANIC: %s\n", e)
	}
	return str
}

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

func newLogger(name string, logMaxSizeInMB int, maxBackups int, maxAgeInDays int) (*log.Logger, *zilorot.Logger) {
	e, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("error opening file: %v", err))
		os.Exit(1)
	}
	logg := log.New(e, "", 0)

	var output *zilorot.Logger

	if logg != nil {
		output = &zilorot.Logger{
			Filename:   name,
			MaxSize:    logMaxSizeInMB,
			MaxBackups: maxBackups,
			MaxAge:     maxAgeInDays,
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

func GetLoggerBySuffix(suffix string, name string, logMaxSizeInMB int, maxBackups int, maxAgeInDays int, writeSource bool) *Logger {
	return NewLogger(name+suffix, logMaxSizeInMB, maxBackups, maxAgeInDays, writeSource)
}
