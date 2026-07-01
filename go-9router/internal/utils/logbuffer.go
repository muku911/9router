package utils

import (
	"regexp"
	"strings"
	"sync"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type LogBuffer struct {
	mu          sync.RWMutex
	lines       []string
	maxLines    int
	subscribers map[chan string]struct{}
	subMu       sync.Mutex
}

func NewLogBuffer(maxLines int) *LogBuffer {
	if maxLines <= 0 {
		maxLines = 500 // default max lines
	}
	return &LogBuffer{
		lines:       make([]string, 0, maxLines),
		maxLines:    maxLines,
		subscribers: make(map[chan string]struct{}),
	}
}

// Strip ANSI escape sequences from a string
func stripAnsi(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

// Write implements the io.Writer interface
func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	lb.mu.Lock()

	raw := string(p)
	clean := stripAnsi(raw)

	// If there are multiple newlines in the input, split them into individual log entries
	// Note: log.Println usually appends a single newline at the end.
	newLines := strings.Split(clean, "\n")

	var added []string
	for _, line := range newLines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" && len(newLines) > 1 {
			// Skip empty lines if splitting resulted in a trailing empty string
			continue
		}

		lb.lines = append(lb.lines, trimmed)
		added = append(added, trimmed)
	}

	// Keep slice size within maxLines bounds
	if len(lb.lines) > lb.maxLines {
		lb.lines = lb.lines[len(lb.lines)-lb.maxLines:]
	}

	lb.mu.Unlock()

	// Broadcast new lines to SSE subscribers (non-blocking)
	lb.subMu.Lock()
	for _, line := range added {
		for ch := range lb.subscribers {
			select {
			case ch <- line:
			default:
				// Drop if subscriber is slow
			}
		}
	}
	lb.subMu.Unlock()

	return len(p), nil
}

// GetLogs returns a copy of the captured log lines
func (lb *LogBuffer) GetLogs() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Return a copy to prevent race conditions during iteration
	logs := make([]string, len(lb.lines))
	copy(logs, lb.lines)
	return logs
}

// ClearLogs empties the captured log lines and notifies SSE subscribers
func (lb *LogBuffer) ClearLogs() {
	lb.mu.Lock()
	lb.lines = make([]string, 0, lb.maxLines)
	lb.mu.Unlock()

	// Notify subscribers of clear event
	lb.subMu.Lock()
	for ch := range lb.subscribers {
		select {
		case ch <- "\x00CLEAR\x00": // sentinel value for clear event
		default:
		}
	}
	lb.subMu.Unlock()
}

// Subscribe returns a channel that receives new log lines and a cancel function.
// The channel also receives "\x00CLEAR\x00" when logs are cleared.
func (lb *LogBuffer) Subscribe() (chan string, func()) {
	ch := make(chan string, 64)
	lb.subMu.Lock()
	lb.subscribers[ch] = struct{}{}
	lb.subMu.Unlock()

	cancel := func() {
		lb.subMu.Lock()
		delete(lb.subscribers, ch)
		lb.subMu.Unlock()
	}

	return ch, cancel
}
