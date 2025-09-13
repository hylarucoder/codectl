package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// simple line reader with history-like UX
type liner struct{ r *bufio.Reader }

func newLiner() *liner        { return &liner{r: bufio.NewReader(os.Stdin)} }
func (l *liner) Close() error { return nil }
func (l *liner) Prompt(prompt string) (string, error) {
	fmt.Print(prompt)
	s, err := l.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

// pickFromList parses a selection string and returns chosen items from src.
// Accepts empty -> all, "none" -> empty, comma-separated indices (1-based) or names.
func pickFromList(sel string, src []string) []string {
	s := strings.TrimSpace(strings.ToLower(sel))
	if s == "" { // default to all
		out := make([]string, len(src))
		copy(out, src)
		return out
	}
	if s == "none" || s == "no" || s == "0" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	picked := map[string]bool{}
	// map names for quick lookup
	nameIdx := map[string]string{}
	for _, v := range src {
		nameIdx[strings.ToLower(strings.TrimSpace(v))] = v
	}
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		// try as index (1-based)
		if n, err := strconv.Atoi(t); err == nil {
			if n >= 1 && n <= len(src) {
				picked[src[n-1]] = true
				continue
			}
		}
		// try as name
		if v, ok := nameIdx[strings.ToLower(t)]; ok {
			picked[v] = true
		}
	}
	out := make([]string, 0, len(picked))
	for k := range picked {
		out = append(out, k)
	}
	// preserve input order; if none matched, default to all
	if len(out) == 0 {
		out = make([]string, len(src))
		copy(out, src)
	}
	return out
}
