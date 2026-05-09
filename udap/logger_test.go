package udap

import (
	"bytes"
	"log"
	"os"
	"testing"
)

func TestNewStructuredLoggerWritesToStderr(t *testing.T) {
	l := NewStructuredLogger()
	if l.logger.Writer() != os.Stderr {
		t.Fatalf("expected logger writer to be os.Stderr, got %v", l.logger.Writer())
	}
}

func TestStructuredLoggerLogsMessageWithFields(t *testing.T) {
	var buf bytes.Buffer
	l := &StructuredLogger{
		level:  LogLevelInfo,
		logger: log.New(&buf, "", 0),
	}
	l.Info("hello", "k", "v")

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("hello")) {
		t.Fatalf("expected output to contain 'hello', got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("k=v")) {
		t.Fatalf("expected output to contain 'k=v', got %q", got)
	}
}
