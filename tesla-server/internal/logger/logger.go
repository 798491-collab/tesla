package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	InfoLog  *log.Logger
	WarnLog  *log.Logger
	ErrorLog *log.Logger
	logDir   string
	logFile  *os.File
)

func Init(dir string) error {
	logDir = dir
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	filename := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	logFile = f

	multiWriter := io.MultiWriter(os.Stdout, f)

	InfoLog = log.New(multiWriter, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	WarnLog = log.New(multiWriter, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(multiWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)

	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return nil
}

func Rotate() {
	if logFile == nil {
		return
	}

	filename := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	currentName := logFile.Name()

	if filename == currentName {
		return
	}

	newFile, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}

	multiWriter := io.MultiWriter(os.Stdout, newFile)

	log.SetOutput(multiWriter)
	if InfoLog != nil {
		InfoLog.SetOutput(multiWriter)
	}
	if WarnLog != nil {
		WarnLog.SetOutput(multiWriter)
	}
	if ErrorLog != nil {
		ErrorLog.SetOutput(multiWriter)
	}

	oldFile := logFile
	logFile = newFile
	oldFile.Close()
}

func Close() {
	if logFile != nil {
		logFile.Close()
	}
}
