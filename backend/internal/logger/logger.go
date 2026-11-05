// Package logger installs the slog handler of the application. Every record goes to the console
// (level colorized), to a daily rotating od-YYYY-MM-DD.log file kept for 14 days (one JSON object
// per line) and, once SetDatabase was called, to the public.log table.
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const maxLogFileAgeDays = 14

// Execer is the part of the database the log sink needs
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
}

var (
	fileMutex       sync.Mutex
	currentLogFile  *os.File
	currentFileDate string

	databaseMutex sync.RWMutex
	database      Execer
)

// Setup makes the handler of this package the default slog logger
func Setup() {
	slog.SetDefault(slog.New(&handler{}))
}

// SetDatabase starts writing the records to the public.log table (nil stops it)
func SetDatabase(execer Execer) {
	databaseMutex.Lock()
	defer databaseMutex.Unlock()
	database = execer
}

type handler struct {
	attrs []slog.Attr
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}

func (h *handler) Handle(_ context.Context, record slog.Record) error {
	level := strings.ToLower(record.Level.String())
	message := h.format(record)

	fmt.Println(colorize(level) + ": " + message)
	writeToFile(level, message, record.Time)

	databaseMutex.RLock()
	execer := database
	databaseMutex.RUnlock()
	if execer != nil {
		go writeToDatabase(execer, level, message)
	}
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{attrs: append(slices.Clone(h.attrs), attrs...)}
}

// WithGroup is accepted but the groups are flattened: the log table has one message column
func (h *handler) WithGroup(string) slog.Handler {
	return h
}

// format renders "message key=value key=value"
func (h *handler) format(record slog.Record) string {
	var builder strings.Builder
	builder.WriteString(record.Message)
	write := func(attr slog.Attr) bool {
		builder.WriteString(" ")
		builder.WriteString(attr.Key)
		builder.WriteString("=")
		builder.WriteString(attr.Value.String())
		return true
	}
	for _, attr := range h.attrs {
		write(attr)
	}
	record.Attrs(write)
	return builder.String()
}

func colorize(level string) string {
	switch level {
	case "error":
		return "\033[31m" + level + "\033[39m"
	case "warn":
		return "\033[33m" + level + "\033[39m"
	case "info":
		return "\033[32m" + level + "\033[39m"
	default:
		return level
	}
}

func writeToFile(level string, message string, at time.Time) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	timestamp := at.UTC().Format("2006-01-02T15:04:05.000Z")
	today := timestamp[:10]
	if currentLogFile == nil || currentFileDate != today {
		rotateLogFile(today)
	}
	if currentLogFile == nil {
		return
	}
	line, _ := json.Marshal(map[string]string{"level": level, "message": message, "timestamp": timestamp})
	currentLogFile.Write(append(line, '\n'))
}

func rotateLogFile(today string) {
	if currentLogFile != nil {
		currentLogFile.Close()
		currentLogFile = nil
	}
	file, err := os.OpenFile("od-"+today+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[logger] failed to open log file: "+err.Error())
		return
	}
	currentLogFile = file
	currentFileDate = today
	deleteOldLogFiles()
}

func deleteOldLogFiles() {
	oldestAllowed := time.Now().AddDate(0, 0, -maxLogFileAgeDays)
	logFiles, _ := filepath.Glob("od-*.log")
	for _, logFile := range logFiles {
		dateText := strings.TrimSuffix(strings.TrimPrefix(logFile, "od-"), ".log")
		fileDate, err := time.Parse("2006-01-02", dateText)
		if err == nil && fileDate.Before(oldestAllowed) {
			os.Remove(logFile)
		}
	}
}

func writeToDatabase(execer Execer, level string, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := execer.Exec(ctx, "INSERT INTO log (date, level, message) VALUES (now(), $1, $2)", level, message); err != nil {
		// Straight to stderr: logging this through slog would try the database again
		fmt.Fprintln(os.Stderr, "[logger] failed to save log to database: "+err.Error())
	}
}
