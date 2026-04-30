package translator

import (
	"fmt"
)

type AlignmentResult struct {
	SourceText    string
	TargetText    string
	SourceTiming []Timestamp
	TargetTiming  []Timestamp
}

type Timestamp struct {
	Start float64
	End   float64
}

type Aligner struct{}

func NewAligner() *Aligner {
	return &Aligner{}
}

func (a *Aligner) Align(transcript *Transcript, translations []string) ([]Segment, error) {
	if len(transcript.Segments) != len(translations) {
		return nil, fmt.Errorf("segments count mismatch: %d vs %d", len(transcript.Segments), len(translations))
	}

	result := make([]Segment, len(transcript.Segments))
	for i, seg := range transcript.Segments {
		result[i] = Segment{
			Index:    seg.Index,
			Start:    seg.Start,
			End:      seg.End,
			Original: seg.Original,
			Final:    translations[i],
		}
	}

	return result, nil
}

func (a *Aligner) AlignBySimilarity(transcript *Transcript, translations []string) ([]Segment, error) {
	result := make([]Segment, len(transcript.Segments))

	for i, seg := range transcript.Segments {
		targetText := ""
		if i < len(translations) {
			targetText = translations[i]
		}

		result[i] = Segment{
			Index:    seg.Index,
			Start:    seg.Start,
			End:      seg.End,
			Original: seg.Original,
			Final:    targetText,
		}
	}

	return result, nil
}