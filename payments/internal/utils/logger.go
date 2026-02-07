package utils

import (
	"encoding/json"
	"log"
	"time"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log("info", msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log("error", msg, fields)
}

func (l *Logger) log(level, msg string, fields map[string]interface{}) {
	entry := map[string]interface{}{
		"level":     level,
		"message":   msg,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range fields {
		entry[k] = v
	}
	b, err := json.Marshal(entry)
	if err != nil {
		log.Printf("level=%s msg=%s", level, msg)
		return
	}
	log.Println(string(b))
}
