package translator

import (
	"fmt"
	"strings"
)

type SRTGenerator struct{}

func NewSRTGenerator() *SRTGenerator {
	return &SRTGenerator{}
}

func (g *SRTGenerator) Generate(segments []Segment) string {
	var builder strings.Builder

	for i, seg := range segments {
		builder.WriteString(fmt.Sprintf("%d\n", i+1))
		builder.WriteString(fmt.Sprintf("%s --> %s\n", formatTimestamp(seg.Start), formatTimestamp(seg.End)))
		builder.WriteString(fmt.Sprintf("%s\n\n", seg.Final))
	}

	return builder.String()
}

func (g *SRTGenerator) GenerateWithOriginal(segments []Segment) string {
	var builder strings.Builder

	for i, seg := range segments {
		builder.WriteString(fmt.Sprintf("%d\n", i+1))
		builder.WriteString(fmt.Sprintf("%s --> %s\n", formatTimestamp(seg.Start), formatTimestamp(seg.End)))
		builder.WriteString(fmt.Sprintf("%s\n", seg.Original))
		builder.WriteString(fmt.Sprintf("%s\n\n", seg.Final))
	}

	return builder.String()
}

func formatTimestamp(seconds float64) string {
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	ms := int((seconds - float64(int(seconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

func ParseTimestamp(ts string) (float64, error) {
	var h, m, ms int
	var s int
	_, err := fmt.Sscanf(ts, "%02d:%02d:%02d,%03d", &h, &m, &s, &ms)
	if err != nil {
		_, err = fmt.Sscanf(ts, "%02d:%02d:%02d.%03d", &h, &m, &s, &ms)
		if err != nil {
			return 0, err
		}
	}

	total := float64(h)*3600 + float64(m)*60 + float64(s) + float64(ms)/1000
	return total, nil
}

type ASSGenerator struct {
	style string
}

func NewASSGenerator() *ASSGenerator {
	return &ASSGenerator{
		style: DefaultStyle(),
	}
}

func DefaultStyle() string {
	return `[Script Info]
Title: Default
ScriptType: v4.00+
WrapStyle: 0
PlayResX: 1920
PlayResY: 1080
ScaledBorderAndShadow: yes

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,48,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text`
}

func (g *ASSGenerator) Generate(segments []Segment) string {
	var builder strings.Builder
	builder.WriteString(g.style)
	builder.WriteString("\n")

	for _, seg := range segments {
		start := formatTimestampASS(seg.Start)
		end := formatTimestampASS(seg.End)
		text := escapeASS(seg.Final)

		builder.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n",
			start, end, text))
	}

	return builder.String()
}

func formatTimestampASS(seconds float64) string {
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	cs := int((seconds - float64(int(seconds))) * 100)

	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

func escapeASS(text string) string {
	text = strings.ReplaceAll(text, "{", "\\{")
	text = strings.ReplaceAll(text, "}", "\\}")
	text = strings.ReplaceAll(text, "\\", "\\\\")
	return text
}