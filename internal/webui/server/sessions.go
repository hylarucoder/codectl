package server

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type sessionMessage struct {
	ID      string    `json:"id"`
	Role    string    `json:"role"` // user|assistant|system
	Content string    `json:"content"`
	Ts      time.Time `json:"ts"`
}

type session struct {
	ID       string           `json:"id"`
	Title    string           `json:"title"`
	Created  time.Time        `json:"created"`
	Messages []sessionMessage `json:"messages"`
}

var (
	sessMu   sync.RWMutex
	sessions = map[string]*session{}
	// per-session subscribers for SSE
	sseMu   sync.RWMutex
	sseSubs = map[string]map[chan string]struct{}{}
)

func newID() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 36) + fmt.Sprintf("%04x", rand.Intn(65536))
}

func sessionsRootHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessMu.RLock()
		list := make([]*session, 0, len(sessions))
		for _, s := range sessions {
			list = append(list, s)
		}
		sessMu.RUnlock()
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var in struct {
			Title string `json:"title"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if strings.TrimSpace(in.Title) == "" {
			in.Title = "Session " + time.Now().Format("01-02 15:04:05")
		}
		s := &session{ID: newID(), Title: in.Title, Created: time.Now()}
		sessMu.Lock()
		sessions[s.ID] = s
		sessMu.Unlock()
		writeJSON(w, http.StatusCreated, s)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// /api/sessions/{id}/... multiplexer
func sessionItemHandler(w http.ResponseWriter, r *http.Request) {
	// path: /api/sessions/{id} or /api/sessions/{id}/xxx
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id := parts[0]
	tail := parts[1:]
	sessMu.RLock()
	s := sessions[id]
	sessMu.RUnlock()
	if s == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(tail) == 0 {
		// GET session detail
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, s)
		return
	}
	switch tail[0] {
	case "messages":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var in struct{ Role, Content string }
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, errJSON(err))
			return
		}
		if in.Role == "" {
			in.Role = "user"
		}
		msg := sessionMessage{ID: newID(), Role: in.Role, Content: in.Content, Ts: time.Now()}
		sessMu.Lock()
		s.Messages = append(s.Messages, msg)
		sessMu.Unlock()
		// notify SSE subscribers
		broadcastSSE(id, sseEvent{"message", msg})
		writeJSON(w, http.StatusCreated, msg)
	case "stream":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		serveSSE(w, r, id)
	case "commands":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var in struct {
			Name string `json:"name"`
			Args any    `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, errJSON(err))
			return
		}
		// MVP: just echo a status event and finish
		broadcastSSE(id, sseEvent{"status", map[string]any{"state": "started", "command": in.Name}})
		time.AfterFunc(400*time.Millisecond, func() { broadcastSSE(id, sseEvent{"status", map[string]any{"state": "done", "command": in.Name}}) })
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// SSE minimal
type sseEvent struct {
	Type string
	Data any
}

func serveSSE(w http.ResponseWriter, r *http.Request, sid string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ch := make(chan string, 8)
	addSSE(sid, ch)
	defer removeSSE(sid, ch)
	// send hello
	fmt.Fprintf(w, "event: status\n")
	fmt.Fprintf(w, "data: {\"state\":\"connected\"}\n\n")
	flusher.Flush()
	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case line := <-ch:
			io.WriteString(w, line)
			io.WriteString(w, "\n\n")
			flusher.Flush()
		}
	}
}

func addSSE(id string, ch chan string) {
	sseMu.Lock()
	if sseSubs[id] == nil {
		sseSubs[id] = map[chan string]struct{}{}
	}
	sseSubs[id][ch] = struct{}{}
	sseMu.Unlock()
}
func removeSSE(id string, ch chan string) {
	sseMu.Lock()
	if subs := sseSubs[id]; subs != nil {
		delete(subs, ch)
	}
	sseMu.Unlock()
}

func broadcastSSE(id string, ev sseEvent) {
	payload, _ := json.Marshal(ev.Data)
	line := fmt.Sprintf("event: %s\ndata: %s", ev.Type, string(payload))
	sseMu.RLock()
	subs := sseSubs[id]
	for c := range subs {
		select {
		case c <- line:
		default:
		}
	}
	sseMu.RUnlock()
}
