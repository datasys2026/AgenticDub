package hitl

import (
	"regexp"
	"strings"
)

var punctuationRegex = regexp.MustCompile(`[，。？！、；：""''（）【】《》「」『』!?,.\:\;\"\'\(\)\[\]\<\>]+`)

func CleanPunctuation(text string) string {
	result := punctuationRegex.ReplaceAllString(text, " ")

	result = strings.ReplaceAll(result, "'", "")
	result = strings.ReplaceAll(result, "\"", "")

	parts := strings.Fields(result)
	result = strings.Join(parts, " ")

	result = strings.TrimRight(result, " ")

	return result
}