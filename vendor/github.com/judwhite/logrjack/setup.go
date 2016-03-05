package logrjack

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Settings specifies the options used to initialize Logrus and Lumberjack.
type Settings struct {
	// WriteStdout determines if the log will be written to Stdout. The default
	// is to only write to the log file. When running as a Windows Service make
	// sure not to write to Stdout.
	WriteStdout bool

	// Filename is the file to write logs to. Backup log files will be retained
	// in the same directory. It uses <processpath>/logs/<processname>.log if empty.
	Filename string `json:"filename" yaml:"filename"`

	// MaxSizeMB is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSizeMB int `json:"maxsize" yaml:"maxsize"`

	// MaxAgeDays is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename. Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAgeDays int `json:"maxage" yaml:"maxage"`

	// MaxBackups is the maximum number of old log files to retain. The default
	// is to retain all old log files (though MaxAgeDays may still cause them to get
	// deleted.)
	MaxBackups int `json:"maxbackups" yaml:"maxbackups"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time. The default is to use UTC
	// time.
	LocalTime bool `json:"localtime" yaml:"localtime"`
}

// Setup initializes Logrus (structured logged) and Lumberjack (log rolling)
func Setup(settings Settings) {
	if settings.Filename == "" {
		settings.Filename = getDefaultLogFilename(os.Args[0])
	}

	fileLog := &lumberjack.Logger{
		Filename:   settings.Filename,
		MaxSize:    settings.MaxSizeMB,
		MaxAge:     settings.MaxAgeDays,
		MaxBackups: settings.MaxBackups,
		LocalTime:  settings.LocalTime,
	}

	logrus.SetFormatter(&logrus.TextFormatter{})

	if settings.WriteStdout {
		multi := io.MultiWriter(os.Stdout, fileLog)
		logrus.SetOutput(multi)
	} else {
		logrus.SetOutput(fileLog)
	}
}

func getDefaultLogFilename(processPath string) string {
	logDir := filepath.Join(filepath.Dir(processPath), "logs")

	logFile := filepath.Base(processPath)
	ext := filepath.Ext(logFile)
	if ext != "" {
		logFile = strings.TrimSuffix(logFile, ext)
	}
	return filepath.Join(logDir, logFile) + ".log"
}
