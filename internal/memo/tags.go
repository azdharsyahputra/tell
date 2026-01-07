package memo

import (
	"regexp"
	"strings"
)

var hashtagRe = regexp.MustCompile(`#([a-zA-Z0-9_]{1,32})`)

func ExtractTags(content string) []string {
	matches := hashtagRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		t := strings.ToLower(m[1])
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)

		if len(out) >= 20 { // cap
			break
		}
	}

	return out
}
