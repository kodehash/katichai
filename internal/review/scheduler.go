package review

import "github.com/katichai/katich/internal/llm"

// ChunkWindow groups chunks that can be sent within a single minute window
// without exceeding the tokens-per-minute (TPM) rate limit.
type ChunkWindow struct {
	Chunks      []*ReviewChunk
	TotalTokens int
}

// scheduleChunkWindows bins chunks into sequential 60-second windows so that
// each window's total estimated token usage stays at or below tpmLimit.
//
// Estimation per chunk:
//
//	input  = chunk.TokenCount (already computed by the chunker)
//	output = min(4096, chunk.TokenCount * 40 / 100)  — conservative 40% heuristic
//	total  = input + output
//
// If tpmLimit == 0 all chunks are returned in a single window, which means the
// caller's existing behaviour is preserved unchanged.
func scheduleChunkWindows(chunks []*ReviewChunk, tpmLimit int) []ChunkWindow {
	if tpmLimit <= 0 || len(chunks) == 0 {
		// No rate limiting — one window containing everything.
		return []ChunkWindow{{Chunks: chunks, TotalTokens: totalEstimate(chunks)}}
	}

	var windows []ChunkWindow
	current := ChunkWindow{}

	for _, c := range chunks {
		est := estimateChunkTokens(c)

		// A single chunk that is already larger than the TPM limit gets its own
		// window — we can't split it further here.
		needsNewWindow := current.TotalTokens+est > tpmLimit && len(current.Chunks) > 0

		if needsNewWindow {
			windows = append(windows, current)
			current = ChunkWindow{}
		}

		current.Chunks = append(current.Chunks, c)
		current.TotalTokens += est
	}

	if len(current.Chunks) > 0 {
		windows = append(windows, current)
	}

	return windows
}

// estimateChunkTokens returns the predicted total token cost for one chunk:
// input tokens already stored on the chunk + a conservative output estimate.
func estimateChunkTokens(c *ReviewChunk) int {
	inputTokens := c.TokenCount
	// Re-add the system-prompt tokens that were excluded from TokenCount but are
	// still billed by the API.
	systemTokens := llm.EstimateTokens(llm.SystemPrompt)
	// Output is capped at 4096 per chunk (same cap used when calling the API).
	outputTokens := inputTokens * 40 / 100
	if outputTokens > 4096 {
		outputTokens = 4096
	}
	return systemTokens + inputTokens + outputTokens
}

// totalEstimate sums estimated tokens across all chunks (used for the no-limit case).
func totalEstimate(chunks []*ReviewChunk) int {
	total := 0
	for _, c := range chunks {
		total += estimateChunkTokens(c)
	}
	return total
}

