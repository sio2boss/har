package har

import (
	"io"
	"os"

	"github.com/cheggaaa/pb/v3"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
	silent bool
	bar    *pb.ProgressBar
}

var globalLogger *Logger

func init() {
	globalLogger = NewLogger(false)
}

func NewLogger(silent bool) *Logger {
	l := &Logger{
		Logger: logrus.New(),
		silent: silent,
	}

	// Configure logger
	l.SetOutput(os.Stdout)

	if silent {
		l.SetLevel(logrus.ErrorLevel)
	} else {
		l.SetLevel(logrus.InfoLevel)
	}

	return l
}

func (l *Logger) SetSilent(silent bool) {
	l.silent = silent
	if silent {
		l.SetLevel(logrus.ErrorLevel)
	} else {
		l.SetLevel(logrus.InfoLevel)
	}
}

func (l *Logger) StartProgress(total int64) {
	if !l.silent {
		l.bar = pb.New64(total)
		l.bar.SetWidth(100)
		l.bar.Start()
	}
}

func (l *Logger) StopProgress() {
	if !l.silent && l.bar != nil {
		l.bar.Finish()
		l.bar = nil
	}
}

func (l *Logger) GetProgressReader(reader io.Reader) io.Reader {
	if !l.silent && l.bar != nil {
		return l.bar.NewProxyReader(reader)
	}
	return reader
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	return globalLogger
}

// Fatal logs a fatal error and exits
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// IsSilent returns true if the logger is silent
func (l *Logger) IsSilent() bool {
	return l.silent
}
