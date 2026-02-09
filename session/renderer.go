package session

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/glamour"
)

// StreamingMarkdownRenderer handles incremental markdown rendering
// for streaming LLM responses while maintaining real-time output feel.
type StreamingMarkdownRenderer struct {
	buffer          strings.Builder
	glamourRenderer *glamour.TermRenderer
	inCodeBlock     bool
	codeFenceMarker string
	writer          io.Writer
}

// RendererOption is a functional option for configuring the renderer
type RendererOption func(*StreamingMarkdownRenderer)

// WithWriter sets a custom writer for output (useful for testing)
func WithWriter(w io.Writer) RendererOption {
	return func(r *StreamingMarkdownRenderer) {
		r.writer = w
	}
}

// WithGlamourRenderer sets a custom glamour renderer (useful for testing)
func WithGlamourRenderer(gr *glamour.TermRenderer) RendererOption {
	return func(r *StreamingMarkdownRenderer) {
		r.glamourRenderer = gr
	}
}

// NewStreamingMarkdownRenderer creates a new renderer with dark theme and syntax highlighting
func NewStreamingMarkdownRenderer(opts ...RendererOption) (*StreamingMarkdownRenderer, error) {
	gr, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	r := &StreamingMarkdownRenderer{
		glamourRenderer: gr,
		writer:          nil, // nil means use fmt.Print (stdout)
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// ProcessDelta processes an incoming delta chunk from the streaming response.
// It buffers content and renders complete markdown elements as they are detected.
func (r *StreamingMarkdownRenderer) ProcessDelta(delta string) {
	r.buffer.WriteString(delta)

	// Process the buffer to find and render complete elements
	r.processBuffer()
}

// processBuffer analyzes the buffer and renders complete markdown elements
func (r *StreamingMarkdownRenderer) processBuffer() {
	content := r.buffer.String()

	// Track code block state
	r.updateCodeBlockState(content)

	// If we're inside a code block, don't render until it's complete
	if r.inCodeBlock {
		return
	}

	// Look for complete elements to render
	// We consider content complete when we have:
	// 1. A complete code block (opened and closed)
	// 2. A paragraph followed by a blank line
	// 3. Content ending with double newline

	// Find the last safe render point
	renderPoint := r.findRenderPoint(content)

	if renderPoint > 0 {
		toRender := content[:renderPoint]
		remaining := content[renderPoint:]

		r.renderContent(toRender)

		// Keep remaining content in buffer
		r.buffer.Reset()
		r.buffer.WriteString(remaining)
	}
}

// updateCodeBlockState tracks whether we're currently inside a code block
func (r *StreamingMarkdownRenderer) updateCodeBlockState(content string) {
	lines := strings.Split(content, "\n")

	// Reset state and recompute from buffer content
	r.inCodeBlock = false
	r.codeFenceMarker = ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !r.inCodeBlock {
			// Check for code fence opening
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				r.inCodeBlock = true
				if strings.HasPrefix(trimmed, "```") {
					r.codeFenceMarker = "```"
				} else {
					r.codeFenceMarker = "~~~"
				}
			}
		} else {
			// Check for code fence closing
			if trimmed == r.codeFenceMarker || strings.HasPrefix(trimmed, r.codeFenceMarker) {
				// Only close if the line is just the fence marker (possibly with trailing spaces)
				if trimmed == r.codeFenceMarker {
					r.inCodeBlock = false
					r.codeFenceMarker = ""
				}
			}
		}
	}
}

// findRenderPoint finds the position in content where we can safely render
// Returns 0 if no safe render point is found
func (r *StreamingMarkdownRenderer) findRenderPoint(content string) int {
	// Don't render if we're in a code block
	if r.inCodeBlock {
		return 0
	}

	// Look for double newline (paragraph break) - render everything before it
	if idx := strings.LastIndex(content, "\n\n"); idx != -1 {
		return idx + 2 // Include the double newline
	}

	// Look for single newline at the end after a complete line
	// This helps with streaming line-by-line content
	if strings.HasSuffix(content, "\n") {
		// Check if we have at least one complete line
		lines := strings.Split(content, "\n")
		if len(lines) > 1 {
			// Render all complete lines except the last (which is empty after split)
			lastNewline := strings.LastIndex(content, "\n")
			if lastNewline > 0 {
				return lastNewline + 1
			}
		}
	}

	return 0
}

// renderContent renders the given markdown content using glamour
func (r *StreamingMarkdownRenderer) renderContent(content string) {
	if content == "" {
		return
	}

	rendered, err := r.glamourRenderer.Render(content)
	if err != nil {
		// Fallback to plain text if rendering fails
		r.output(content)
		return
	}

	// Glamour adds extra newlines, trim trailing ones to avoid double spacing
	rendered = strings.TrimSuffix(rendered, "\n")

	r.output(rendered)
}

// output writes content to the configured writer or stdout
func (r *StreamingMarkdownRenderer) output(content string) {
	if r.writer != nil {
		r.writer.Write([]byte(content))
	} else {
		fmt.Print(content)
	}
}

// Flush renders any remaining content in the buffer.
// Should be called when the stream ends (e.g., on session.idle event).
func (r *StreamingMarkdownRenderer) Flush() {
	content := r.buffer.String()
	if content == "" {
		return
	}

	r.renderContent(content)
	r.buffer.Reset()
	r.inCodeBlock = false
	r.codeFenceMarker = ""
}

// Reset clears the buffer and resets state for a new message.
func (r *StreamingMarkdownRenderer) Reset() {
	r.buffer.Reset()
	r.inCodeBlock = false
	r.codeFenceMarker = ""
}

// GetBufferContent returns the current buffer content (useful for testing)
func (r *StreamingMarkdownRenderer) GetBufferContent() string {
	return r.buffer.String()
}

// IsInCodeBlock returns whether the renderer is currently tracking an open code block
func (r *StreamingMarkdownRenderer) IsInCodeBlock() bool {
	return r.inCodeBlock
}
