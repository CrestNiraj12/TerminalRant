package common

import "strings"

// StripHashtag removes the tracked hashtag (e.g. "#terminalrant") from the end of the text.
// It matches Case-Insensitive but preserves the original text's case for the rest.
func StripHashtag(content, hashtag string) string {
	if hashtag == "" {
		return content
	}
	tag := "#" + hashtag
	trimmed := strings.TrimSpace(content)
	if strings.HasSuffix(strings.ToLower(trimmed), strings.ToLower(tag)) {
		return strings.TrimSpace(trimmed[:len(trimmed)-len(tag)])
	}
	return content
}
