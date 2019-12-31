package common

import (
	"io"
	"log"
	"os"
)

var Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

type LogWriter struct {
	name string
	l    *log.Logger
	w    io.Writer
}

func NewLogWriter(name string) *LogWriter {
	return &LogWriter{
		name: name,
		l:    log.New(os.Stdout, "", log.Ldate|log.Ltime),
	}
}

func (w *LogWriter) Write(d []byte) (int, error) {
	w.l.Printf("%s|%s", w.name, string(d))
	return 0, nil
}
