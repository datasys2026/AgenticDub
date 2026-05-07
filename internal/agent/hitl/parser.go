package hitl

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ReviewParser interface {
	Parse(path string) (ReviewDocument, error)
	Generate(doc ReviewDocument) (string, error)
}

type TxtParser struct{}

var (
	segmentHeaderRegex = regexp.MustCompile(`【第 (\d+) 句】 (.+) --> (.+)`)
	originalLineRegex  = regexp.MustCompile(`^原文：(.*)$`)
	subtitleLineRegex   = regexp.MustCompile(`^字幕：(.*)$`)
)

func (p TxtParser) Parse(path string) (ReviewDocument, error) {
	file, err := os.Open(path)
	if err != nil {
		return ReviewDocument{}, err
	}
	defer file.Close()

	var segments []Segment
	var currentIndex int
	var currentStart, currentEnd time.Time
	var currentOriginal, currentEdited string
	state := 0 // 0: waiting for header, 1: in segment

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if matches := segmentHeaderRegex.FindStringSubmatch(line); len(matches) == 4 {
			if state == 1 && currentOriginal != "" {
				segments = append(segments, Segment{
					Index:    currentIndex,
					Start:    currentStart,
					End:      currentEnd,
					Original: currentOriginal,
					Edited:   currentEdited,
				})
			}

			currentIndex, _ = strconv.Atoi(matches[1])
			currentStart, _ = parseTimestamp(matches[2])
			currentEnd, _ = parseTimestamp(matches[3])
			currentOriginal = ""
			currentEdited = ""
			state = 1
			continue
		}

		if matches := originalLineRegex.FindStringSubmatch(line); len(matches) == 2 {
			currentOriginal = matches[1]
			continue
		}

		if matches := subtitleLineRegex.FindStringSubmatch(line); len(matches) == 2 {
			currentEdited = matches[1]
			continue
		}
	}

	if state == 1 && currentOriginal != "" {
		segments = append(segments, Segment{
			Index:    currentIndex,
			Start:    currentStart,
			End:      currentEnd,
			Original: currentOriginal,
			Edited:   currentEdited,
		})
	}

	doc := ReviewDocument{
		Segments: segments,
		CreatedAt: time.Now(),
	}

	return doc, nil
}

func (p TxtParser) Generate(doc ReviewDocument) (string, error) {
	var builder strings.Builder

	for _, seg := range doc.Segments {
		builder.WriteString(fmt.Sprintf("【第 %d 句】 %s\n", seg.Index, seg.TimeRange()))
		builder.WriteString(fmt.Sprintf("原文：%s\n", seg.Original))
		builder.WriteString(fmt.Sprintf("字幕：%s\n", seg.Edited))
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", ":")

	parts := strings.Split(s, ":")
	if len(parts) != 4 {
		return time.Time{}, fmt.Errorf("invalid timestamp: %s", s)
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	seconds, _ := strconv.Atoi(parts[2])
	millis, _ := strconv.Atoi(parts[3])

	return time.Date(0, 1, 1, hours, minutes, seconds, millis*1000000, time.UTC), nil
}

// formatTimestamp is in entity.go