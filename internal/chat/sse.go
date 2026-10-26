package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// sseWriter serializa escrituras SSE y hace flush después de cada evento.
// Es seguro para usar desde el handler y el keepalive concurrentemente.
type sseWriter struct {
	w       io.Writer
	flusher http.Flusher
	mu      sync.Mutex
}

func newSSEWriter(w http.ResponseWriter) (*sseWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("el servidor HTTP no soporta streaming")
	}
	return &sseWriter{w: w, flusher: flusher}, nil
}

func (s *sseWriter) Send(event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializando evento SSE: %w", err)
	}
	event = strings.TrimSpace(event)
	if event == "" {
		event = "message"
	}
	return s.write(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data))
}

func (s *sseWriter) Comment(text string) error {
	text = strings.ReplaceAll(text, "\n", " ")
	return s.write(": " + text + "\n\n")
}

func (s *sseWriter) write(raw string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := io.WriteString(s.w, raw); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}
