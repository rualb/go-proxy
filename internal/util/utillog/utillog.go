package utillog

import (
	"fmt"
	"log"
	"log/slog"
	"os"
)

// var DefaultWriter = bufio.NewWriterSize(os.Stdout, 4096*10) // need mutex

var DefaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// var DefaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
// 	// Level: slog.LevelInfo, // Set the minimum log level to Warning
// }))

// var DefaultLogger = log.New(os.Stdout, "", log.LUTC)

func Info(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	DefaultLogger.Info(msg)
}

func Error(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	DefaultLogger.Error(msg)

}

func Panic(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	DefaultLogger.Error(msg)

	log.Panic(msg)

}
func Debug(format string, v ...any) {

	msg := fmt.Sprintf(format, v...)
	DefaultLogger.Debug(msg)
}
func Warn(format string, v ...any) {

	msg := fmt.Sprintf(format, v...)
	DefaultLogger.Warn(msg)
}
func Sync() {
	// if zap
	fmt.Print("log sync...")
}
