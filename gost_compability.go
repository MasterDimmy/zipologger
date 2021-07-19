package zipologger

import (
	"fmt"
)

/*
	for reproxy-gost
*/

// LogLogger uses the standard log package as the logger
type GostLogger struct {
	L *Logger
}

// Log uses the standard log library log.Output
func (l *GostLogger) Log(v ...interface{}) {
	l.L.Print(fmt.Sprint(v...))
}

// Logf uses the standard log library log.Output
func (l *GostLogger) Logf(format string, v ...interface{}) {
	l.L.Printf(format, v[0], v[1:]...)
}
