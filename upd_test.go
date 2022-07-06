package zipologger

import (
	"testing"
)

var logger *Logger

func Test_writelog(t *testing.T) {
	defer Wait()

	logger = NewLogger("test", 1, 1, 1, false)
	logger.Print("test from main")
	logger.SetAlsoToStdout(true).Print("also to stdout")
	logger.SetAlsoToStdout(false).Print("stdout disabled")

	logger.WriteDateTime(false).Print("no date time")
	logger.WriteDateTime(true).Print("has date time")

	logger.WriteSourcePath(false).Print("no source path")
	logger.WriteSourcePath(true).Print("has source path")
}
