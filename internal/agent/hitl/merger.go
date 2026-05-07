package hitl

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type SRTSegment struct {
	Index int
	Start string
	End   string
	Text  string
}

type MergedSRT struct {
	Segments []SRTSegment
}

type SRTMerger struct{}

func (m SRTMerger) Merge(originalSRTPath string, editedSegments []Segment) (MergedSRT, error) {
	file, err := os.Open(originalSRTPath)
	if err != nil {
		return MergedSRT{}, err
	}
	defer file.Close()

	editMap := make(map[int]string)
	for _, seg := range editedSegments {
		if seg.HasEdit() {
			editMap[seg.Index] = seg.Edited
		}
	}

	var segments []SRTSegment
	var currentIndex int
	var currentStart, currentEnd string
	var currentText string
	state := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			if state == 3 && currentIndex > 0 {
				text := currentText
				if edited, ok := editMap[currentIndex]; ok {
					text = edited
				}
				segments = append(segments, SRTSegment{
					Index: currentIndex,
					Start: currentStart,
					End:   currentEnd,
					Text:  text,
				})
			}
			state = 0
			continue
		}

		if state == 0 {
			if matches := parseIndexLine(line); len(matches) == 2 {
				currentIndex, _ = toInt(matches[1])
				state = 1
			}
		} else if state == 1 {
			if matches := parseTimeLine(line); len(matches) == 3 {
				currentStart = matches[1]
				currentEnd = matches[2]
				state = 2
			}
		} else if state == 2 {
			currentText = line
			state = 3
		}
	}

	if state == 3 && currentIndex > 0 {
		text := currentText
		if edited, ok := editMap[currentIndex]; ok {
			text = edited
		}
		segments = append(segments, SRTSegment{
			Index: currentIndex,
			Start: currentStart,
			End:   currentEnd,
			Text:  text,
		})
	}

	return MergedSRT{Segments: segments}, nil
}

func (m SRTMerger) Write(outputPath string, merged MergedSRT) error {
	var builder strings.Builder

	for _, seg := range merged.Segments {
		builder.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s\n\n", seg.Index, seg.Start, seg.End, seg.Text))
	}

	return os.WriteFile(outputPath, []byte(builder.String()), 0644)
}

var (
	indexLineRegex = regexp.MustCompile(`^(\d+)$`)
	timeLineRegex  = regexp.MustCompile(`^(.{12}) --> (.{12})$`)
)

func parseIndexLine(line string) []string {
	return indexLineRegex.FindStringSubmatch(line)
}

func parseTimeLine(line string) []string {
	return timeLineRegex.FindStringSubmatch(line)
}

func toInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}