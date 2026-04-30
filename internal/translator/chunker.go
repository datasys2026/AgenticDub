package translator

type ChunkerConfig struct {
	MaxChars    int
	MaxSegments int
}

func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		MaxChars:    600,
		MaxSegments: 10,
	}
}

type Chunker struct {
	config ChunkerConfig
}

func NewChunker(config ChunkerConfig) *Chunker {
	if config.MaxChars == 0 {
		config.MaxChars = 600
	}
	if config.MaxSegments == 0 {
		config.MaxSegments = 10
	}
	return &Chunker{config: config}
}

func (c *Chunker) Split(transcript *Transcript) []*Chunk {
	chunks := make([]*Chunk, 0)
	currentChunk := newChunkWithConfig(0, transcript.Language, "", c.config)

	for _, seg := range transcript.Segments {
		currentChunk.AddSegment(seg)

		if currentChunk.IsFull() {
			chunks = append(chunks, currentChunk)
			currentChunk = newChunkWithConfig(len(chunks), transcript.Language, "", c.config)
		}
	}

	if len(currentChunk.Segments) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

func newChunkWithConfig(index int, sourceLang, targetLang string, config ChunkerConfig) *Chunk {
	chunk := NewChunk(index, sourceLang, targetLang)
	chunk.maxChars = config.MaxChars
	chunk.maxSegments = config.MaxSegments
	return chunk
}

type ChunkerOption func(*Chunker)

func WithMaxChars(maxChars int) ChunkerOption {
	return func(c *Chunker) {
		c.config.MaxChars = maxChars
	}
}

func WithMaxSegments(maxSegments int) ChunkerOption {
	return func(c *Chunker) {
		c.config.MaxSegments = maxSegments
	}
}

func NewChunkerWithOptions(options ...ChunkerOption) *Chunker {
	chunker := &Chunker{config: DefaultChunkerConfig()}
	for _, opt := range options {
		opt(chunker)
	}
	return chunker
}