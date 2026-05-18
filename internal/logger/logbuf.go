package logger

import (
	"fmt"
	"strings"
	"sync"
)

// LogBuffer is a thread-safe ring buffer for recent log entries.
type LogBuffer struct {
	mu      sync.RWMutex
	entries []LogEntryMsg
	head    int
	size    int
	count   int
}

// LogEntryMsg is a simplified log entry for API consumption.
type LogEntryMsg struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Attrs     string `json:"attrs,omitempty"`
}

// RecentLogs returns a copy of the buffered entries in chronological order.
func (b *LogBuffer) RecentLogs() []LogEntryMsg {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]LogEntryMsg, 0, b.count)
	for i := 0; i < b.count; i++ {
		idx := (b.head + b.size - b.count + i) % b.size
		out = append(out, b.entries[idx])
	}
	return out
}

func (b *LogBuffer) append(e LogEntryMsg) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.size == 0 {
		return
	}
	b.entries[b.head] = e
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	}
}

var globalLogBuffer = &LogBuffer{
	entries: make([]LogEntryMsg, 500),
	size:    500,
}

// GlobalLogBuffer returns the global ring buffer.
func GlobalLogBuffer() *LogBuffer {
	return globalLogBuffer
}

// CaptureLogToBuffer configures the default handler to also capture
// log entries into the global ring buffer.
// CaptureLogToBuffer wraps the current ConsumeFunc so that log entries
// are also captured into the global ring buffer. Safe to call after
// SetConsumeFunc — it chains, not replaces.
func CaptureLogToBuffer() {
	prev := defaultHandler.consume.load()
	defaultHandler.consume.store(func(entries []LogEntry) []LogEntry {
		for _, e := range entries {
			var attrBuf strings.Builder
			for _, a := range e.Attrs {
				if a.Key != "" {
					attrBuf.WriteString(a.Key)
					attrBuf.WriteString("=")
					attrBuf.WriteString(fmt.Sprint(a.Value.Any()))
					attrBuf.WriteString(" ")
				}
			}
			globalLogBuffer.append(LogEntryMsg{
				Timestamp: e.Timestamp.Format("15:04:05"),
				Level:     e.Level.String(),
				Message:   e.Message,
				Attrs:     strings.TrimSpace(attrBuf.String()),
			})
		}
		if prev != nil {
			return prev(entries)
		}
		return entries
	})
}
