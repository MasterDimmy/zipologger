package zipologger

import (
	"fmt"
)

/*
	for reproxy-gost
*/

// LogLogger uses the standard log package as the logger
type GostLogger struct {
	log *Logger
}

// Log uses the standard log library log.Output
func (l *GostLogger) Log(v ...interface{}) {
	l.log.Print(fmt.Sprint(v...))
}

// Logf uses the standard log library log.Output
func (l *GostLogger) Logf(format string, v ...interface{}) {
	l.log.Printf(format, v[0], v[1:]...)
}
