package tools

import (
    "regexp"
    "strings"
)

var verRe = regexp.MustCompile(`(?i)\bv?(\d+\.\d+\.\d+(?:[\w\.-]+)?)\b`)

func ParseVersion(s string) string {
    s = strings.TrimSpace(s)
    if s == "" {
        return ""
    }
    // Take first line
    line := strings.Split(s, "\n")[0]
    if m := verRe.FindStringSubmatch(line); len(m) > 1 {
        return m[1]
    }
    // Fallback: try on full string
    if m := verRe.FindStringSubmatch(s); len(m) > 1 {
        return m[1]
    }
    return ""
}

// VersionLess compares two semantic versions (best-effort).
// Returns true if a < b.
func VersionLess(a, b string) bool {
    a = NormalizeVersion(a)
    b = NormalizeVersion(b)
    if a == "" || b == "" {
        return false
    }
    as := strings.SplitN(a, "-", 2)[0]
    bs := strings.SplitN(b, "-", 2)[0]
    ap := strings.Split(as, ".")
    bp := strings.Split(bs, ".")
    // pad to length 3
    for len(ap) < 3 {
        ap = append(ap, "0")
    }
    for len(bp) < 3 {
        bp = append(bp, "0")
    }
    for i := 0; i < 3; i++ {
        av := atoiSafe(ap[i])
        bv := atoiSafe(bp[i])
        if av < bv {
            return true
        }
        if av > bv {
            return false
        }
    }
    // If numeric parts equal, pre-release is considered lower than release
    ahasPre := strings.Contains(a, "-")
    bhasPre := strings.Contains(b, "-")
    if ahasPre && !bhasPre {
        return true
    }
    return false
}

func NormalizeVersion(v string) string {
    v = strings.TrimSpace(v)
    v = strings.TrimPrefix(v, "v")
    return v
}

func atoiSafe(s string) int {
    n := 0
    for _, r := range s {
        if r < '0' || r > '9' {
            break
        }
        n = n*10 + int(r-'0')
    }
    return n
}

