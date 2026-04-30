package translator

import (
	"testing"
)

func TestChunker_Split_SingleChunk(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	transcript := &Transcript{
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello"},
			{Index: 1, Start: 2.0, End: 4.0, Original: "World"},
		},
		Language: "en",
	}

	chunks := chunker.Split(transcript)

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
	if len(chunks[0].Segments) != 2 {
		t.Errorf("expected 2 segments in chunk, got %d", len(chunks[0].Segments))
	}
}

func TestChunker_Split_MultipleChunks(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{MaxChars: 10, MaxSegments: 2})

	transcript := &Transcript{
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 1.0, Original: "One"},
			{Index: 1, Start: 1.0, End: 2.0, Original: "Two"},
			{Index: 2, Start: 2.0, End: 3.0, Original: "Three"},
			{Index: 3, Start: 3.0, End: 4.0, Original: "Four"},
		},
		Language: "en",
	}

	chunks := chunker.Split(transcript)

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[0].Segments) != 2 {
		t.Errorf("expected 2 segments in first chunk, got %d", len(chunks[0].Segments))
	}
	if len(chunks[1].Segments) != 2 {
		t.Errorf("expected 2 segments in second chunk, got %d", len(chunks[1].Segments))
	}
}

func TestChunker_Split_ByChars(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{MaxChars: 15, MaxSegments: 10})

	transcript := &Transcript{
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Short"},
			{Index: 1, Start: 2.0, End: 4.0, Original: "Medium length"},
			{Index: 2, Start: 4.0, End: 6.0, Original: "This is a longer segment"},
		},
		Language: "en",
	}

	chunks := chunker.Split(transcript)

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
}

func TestChunker_Split_EmptyTranscript(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	transcript := &Transcript{
		Segments: []Segment{},
		Language: "en",
	}

	chunks := chunker.Split(transcript)

	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(chunks))
	}
}

func TestChunker_Split_PreservesIndices(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{MaxChars: 10, MaxSegments: 1})

	transcript := &Transcript{
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 1.0, Original: "First"},
			{Index: 1, Start: 1.0, End: 2.0, Original: "Second"},
			{Index: 2, Start: 2.0, End: 3.0, Original: "Third"},
		},
		Language: "en",
	}

	chunks := chunker.Split(transcript)

	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("expected chunk index %d, got %d", i, chunk.Index)
		}
	}
}

func TestChunker_WithOptions(t *testing.T) {
	chunker := NewChunkerWithOptions(
		WithMaxChars(500),
		WithMaxSegments(5),
	)

	if chunker.config.MaxChars != 500 {
		t.Errorf("expected MaxChars 500, got %d", chunker.config.MaxChars)
	}
	if chunker.config.MaxSegments != 5 {
		t.Errorf("expected MaxSegments 5, got %d", chunker.config.MaxSegments)
	}
}