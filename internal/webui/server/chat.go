package server

import (
    "bufio"
    "encoding/json"
    "net/http"
    "time"
)

// chatHandler implements a minimal AI SDK UI message stream (SSE) endpoint at POST /api/chat.
// It streams JSON SSE lines (data: { ... }) that the DefaultChatTransport parses on the client.
//
// Request body (subset):
// {
//   "id": "chat_123",
//   "trigger": "submit-message" | "regenerate-message",
//   "messageId": "optional",
//   "messages": [ { id, role, parts: [...] }, ...],
//   ... any extra fields like model, webSearch ...
// }
//
// Response: SSE with events like start/start-step/text-(start|delta|end)/finish-step/finish.
func chatHandler(w http.ResponseWriter, r *http.Request) {
    // Decode but we don't strictly validate fields for this mock.
    // Keep compatibility even if the UI sends extra fields.
    var _payload map[string]any
    _ = json.NewDecoder(r.Body).Decode(&_payload)

    // Prepare SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("x-vercel-ai-ui-message-stream", "v1")
    // Disable certain reverse proxy buffering if present
    w.Header().Set("X-Accel-Buffering", "no")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming unsupported", http.StatusInternalServerError)
        return
    }

    // Buffered writer for efficiency + explicit flushes
    bw := bufio.NewWriter(w)
    defer bw.Flush()

    write := func(v any) {
        b, _ := json.Marshal(v)
        // SSE line: data: <json>\n\n
        _, _ = bw.WriteString("data: ")
        _, _ = bw.Write(b)
        _, _ = bw.WriteString("\n\n")
        _ = bw.Flush()
        flusher.Flush()
    }

    // Stream a minimal sequence
    write(map[string]any{"type": "start"})
    write(map[string]any{"type": "start-step"})

    // Optional: small reasoning prelude
    write(map[string]any{"type": "reasoning-start", "id": "r1"})
    write(map[string]any{"type": "reasoning-delta", "id": "r1", "delta": "Thinking about the response..."})
    write(map[string]any{"type": "reasoning-end", "id": "r1"})

    // Text streaming
    write(map[string]any{"type": "text-start", "id": "text-1"})
    chunks := []string{
        "这是一个 Go 端的 mock 响应，",
        "返回符合 AI SDK UI Message Stream 协议的数据。",
    }
    for _, ch := range chunks {
        write(map[string]any{"type": "text-delta", "id": "text-1", "delta": ch})
        time.Sleep(20 * time.Millisecond)
    }
    write(map[string]any{"type": "text-end", "id": "text-1"})

    // Optional: attach a source link
    write(map[string]any{
        "type":     "source-url",
        "sourceId": "s1",
        "url":      "https://example.com",
        "title":    "Example",
    })

    write(map[string]any{"type": "finish-step"})
    write(map[string]any{"type": "finish"})
}

// chatReconnectHandler answers 204 No Content to indicate there's no ongoing
// stream to resume for this simple mock implementation.
func chatReconnectHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNoContent)
}

