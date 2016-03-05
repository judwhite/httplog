package logrjack

import "github.com/Sirupsen/logrus"

// Info logs an Entry with a status of "info"
func Info(args ...interface{}) {
	logrus.Info(args...)
}

// Infof logs an Entry with a status of "info"
func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

// Warn logs an Entry with a status of "warn"
func Warn(args ...interface{}) {
	logrus.Warn(args...)
}

// Warnf logs an Entry with a status of "warn"
func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

// Error logs an Entry with a status of "error"
func Error(args ...interface{}) {
	logrus.Error(args...)
}

// Errorf logs an Entry with a status of "error"
func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

// Fatal logs an Entry with a status of "fatal" and exits the program
// with status code 1.
func Fatal(args ...interface{}) {
	logrus.Fatal(args...)
}

// Fatalf logs an Entry with a status of "fatal" and exits the program
// with status code 1.
func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}
