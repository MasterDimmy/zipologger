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
	lru "github.com/MasterDimmy/golang-lruexpire"
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
	file           *os.File // Track the underlying file handle for proper cleanup
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

type globalEncryptorType struct {
	m   sync.Mutex
	key *enc.KeyEncrypt
}

var (
	tologCh            = make(chan *loggerMessage, 1000)
	alsoToStdout       bool
	initializedLoggers *lru.Cache
	newLoggerMutex     sync.Mutex
	panicMutex         sync.Mutex
	wMutex             sync.Mutex
	mainGlobalEncryptor = &globalEncryptorType{}
)

func init() {
	initializedLoggers, _ = lru.NewWithEvict(1000, func(key interface{}, value interface{}) {
		if value == nil {
			return
		}
		log := value.(*Logger)
		if log != nil {
			log.Flush()
			if log.zlog != nil {
				log.zlog.Close()
			}
			if log.file != nil {
				log.file.Close()
			}
		}
	})

	go func() {
		defer HandlePanic()

		for elem := range tologCh {
			elem.log.em.Lock()
			if elem.log.log == nil {
				elem.log.log, elem.log.zlog, elem.log.file = newLogger(elem.log.filename, elem.log.logMaxSizeInMB, elem.log.maxBackups, elem.log.maxAgeInDays)
				if elem.log.log == nil {
					// Failed to create logger, skip this message
					elem.log.em.Unlock()
					elem.log.logTasks.Done()
					continue
				}
			}
			elem.log.em.Unlock()

			str := elem.msg

			for strings.HasSuffix(str, "\n") {
				str = strings.TrimSuffix(str, "\n")
			}

			mainGlobalEncryptor.m.Lock()
			elem.log.em.Lock()
			enckey := elem.log.encryptionKey
			if enckey == nil {
				enckey = mainGlobalEncryptor.key
			}
			elem.log.em.Unlock()
			mainGlobalEncryptor.m.Unlock()

			if enckey != nil {
				ret, err := enckey.EncryptString(str)
				if err == nil {
					str = base64.RawStdEncoding.EncodeToString(ret)
				}
			}

			str = str + "\n"
			elem.log.log.Print(str)
			elem.log.logTasks.Done()
		}
	}()
}

// EmptyLogger is a logger that writes nowhere
var EmptyLogger = func() *Logger {
	return &Logger{}
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
	// First check without lock for performance (double-checked locking pattern)
	if logger, ok := initializedLoggers.Get(filename); ok {
		return logger.(*Logger)
	}

	newLoggerMutex.Lock()
	defer newLoggerMutex.Unlock()

	// Check again after acquiring lock
	if logger, ok := initializedLoggers.Get(filename); ok {
		return logger.(*Logger)
	}

	p := filepath.Dir(filename)
	// Use platform-appropriate permissions
	var perm os.FileMode = 0755
	if runtime.GOOS == "windows" {
		perm = 0777 // Windows doesn't use Unix permissions strictly
	}
	if err := os.MkdirAll(p, perm); err != nil {
		// Log error but continue, file creation will fail later if directory doesn't exist
		os.Stderr.WriteString(fmt.Sprintf("Warning: failed to create directory %s: %v\n", p, err))
	}
	
	// Create LRU cache with expiration for limitedPrintf
	l, _ := lru.NewWithExpire(1000, time.Hour*24) // Expire entries after 24 hours
	
	log := &Logger{
		filename:       filename,
		logMaxSizeInMB: logMaxSizeInMB,
		maxBackups:     maxBackups,
		maxAgeInDays:   maxAgeInDays,
		logSourcePath:  writeFileline,
		limitedPrint:   l,
		logDateTime:    true,
	}

	initializedLoggers.Add(filename, log)
	return log
}

func Wait() {
	wMutex.Lock()
	defer wMutex.Unlock()
	
	// Get a snapshot of keys to avoid concurrent modification issues
	keys := initializedLoggers.Keys()
	for _, w := range keys {
		log, ok := initializedLoggers.Get(w)
		if ok && log != nil {
			logger := log.(*Logger)
			if logger != nil {
				logger.Wait()
			}
		}
	}
}

func (l *Logger) GetFileName() string {
	return l.filename
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

// Close explicitly closes the logger's file handle and releases resources.
// This is optional as resources are automatically cleaned up via LRU eviction,
// but can be called explicitly for immediate cleanup.
func (l *Logger) Close() error {
	l.Wait()
	
	l.em.Lock()
	defer l.em.Unlock()
	
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

func (l *Logger) Wait() {
	l.m.Lock()
	if atomic.LoadInt32(&l.waitStarted) == 1 {
		// Already waiting, avoid double wait which could deadlock
		l.m.Unlock()
		return
	}
	atomic.StoreInt32(&l.waitStarted, 1)
	l.m.Unlock()
	
	l.logTasks.Wait()
	
	l.m.Lock()
	atomic.StoreInt32(&l.waitStarted, 0)
	l.m.Unlock()
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

	if len(ret) < 3 {
		if add > 0 {
			ret = formatCaller(0)
		} else {
			ret = "unknown: "
		}
	} else {
		ret = ret + ": "
	}

	if len(ret) > 5 {
		ret += "\n"
	}

	return ret
}

func (l *Logger) print(msg string) string {
	// Early return for EmptyLogger - check if this is the global EmptyLogger instance
	// or if all critical fields are zero values
	if l == EmptyLogger || (l.filename == "" && l.log == nil && l.zlog == nil && l.file == nil) {
		return msg
	}

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

	if alsoToStdout || l.alsoToStdout {
		fmt.Println(msg)
	}

	if l.log != nil {
		select {
		case tologCh <- &loggerMessage{
			msg: msg,
			log: l,
		}:
		default:
			// Channel is full, log to stderr as fallback
			os.Stderr.WriteString("Logger channel full, dropping message: " + msg + "\n")
		}
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
	if len(w) == 0 {
		return l.print("")
	}
	return l.print(fmt.Sprintln(w...))
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
	logsDir := "logs"
	// Use platform-appropriate permissions
	var perm os.FileMode = 0777
	if runtime.GOOS == "windows" {
		perm = 0777
	}
	os.Mkdir(logsDir, perm)
	fn := filepath.Join(filepath.Dir(st), logsDir, "panic_"+filepath.Base(os.Args[0])+time.Now().Format("_2006-Jan-02_15")+".log")
	f, e := os.Create(fn)
	if e != nil {
		// Failed to create panic log file, return empty string
		return ""
	}
	defer f.Close()
	
	_, file, line, _ := runtime.Caller(1)
	str := fmt.Sprintf("Panic in [%s:%d] :\n", file, line) + pdesc + "\nSTACK:\n" + Stack()
	f.WriteString(str)
	return str
}

func newLogger(name string, logMaxSizeInMB int, maxBackups int, maxAgeInDays int) (*log.Logger, *zilorot.Logger, *os.File) {
	e, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		// Return nil values instead of exiting
		os.Stderr.WriteString(fmt.Sprintf("error opening file: %v\n", err))
		return nil, nil, nil
	}
	
	logg := log.New(e, "", 0)
	output := &zilorot.Logger{
		Filename:   name,
		MaxSize:    logMaxSizeInMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAgeInDays,
	}
	logg.SetOutput(output)

	return logg, output, e
}

var ErrorCatcher *errorcatcher.System

func HandlePanic() {
	if e := recover(); e != nil {
		p := fmt.Sprintf("%v", e)
		fmt.Println(p)
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
