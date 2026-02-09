package session

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// ansiRegex matches ANSI escape codes
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// Helper function to create a test renderer with output capture
func createTestRenderer(t *testing.T) (*StreamingMarkdownRenderer, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	r, err := NewStreamingMarkdownRenderer(WithWriter(buf))
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}
	return r, buf
}

// Helper function to stream deltas one at a time
func streamDeltas(r *StreamingMarkdownRenderer, deltas []string) {
	for _, delta := range deltas {
		r.ProcessDelta(delta)
	}
}

// containsText checks if output contains expected text after stripping ANSI codes
func containsText(output, expected string) bool {
	return strings.Contains(stripANSI(output), expected)
}

// =============================================================================
// Test: Basic Renderer Creation
// =============================================================================

func TestNewStreamingMarkdownRenderer(t *testing.T) {
	r, err := NewStreamingMarkdownRenderer()
	if err != nil {
		t.Fatalf("NewStreamingMarkdownRenderer() error = %v", err)
	}
	if r == nil {
		t.Fatal("NewStreamingMarkdownRenderer() returned nil")
	}
	if r.glamourRenderer == nil {
		t.Error("glamour renderer should be initialized")
	}
}

func TestNewStreamingMarkdownRendererWithWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	r, err := NewStreamingMarkdownRenderer(WithWriter(buf))
	if err != nil {
		t.Fatalf("NewStreamingMarkdownRenderer() error = %v", err)
	}
	if r.writer != buf {
		t.Error("writer should be set to provided buffer")
	}
}

// =============================================================================
// Test: Basic Markdown Elements
// =============================================================================

func TestRenderHeader(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("# Hello World\n\n")
	r.Flush()

	output := buf.String()
	// Glamour renders headers with ANSI codes for styling
	// We just verify the text content is present
	if !containsText(output, "Hello World") {
		t.Errorf("expected output to contain 'Hello World', got: %s", output)
	}
}

func TestRenderBoldText(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("This is **bold** text.\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "bold") {
		t.Errorf("expected output to contain 'bold', got: %s", output)
	}
}

func TestRenderInlineCode(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("Use the `fmt.Println` function.\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "fmt.Println") {
		t.Errorf("expected output to contain 'fmt.Println', got: %s", output)
	}
}

func TestRenderListItems(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("- Item 1\n- Item 2\n- Item 3\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "Item 1") {
		t.Errorf("expected output to contain 'Item 1', got: %s", output)
	}
	if !containsText(output, "Item 2") {
		t.Errorf("expected output to contain 'Item 2', got: %s", output)
	}
	if !containsText(output, "Item 3") {
		t.Errorf("expected output to contain 'Item 3', got: %s", output)
	}
}

func TestRenderNumberedList(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("1. First item\n2. Second item\n3. Third item\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "First item") {
		t.Errorf("expected output to contain 'First item', got: %s", output)
	}
	if !containsText(output, "Second item") {
		t.Errorf("expected output to contain 'Second item', got: %s", output)
	}
}

func TestRenderLinks(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("Visit [Go](https://golang.org) for more info.\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "Go") {
		t.Errorf("expected output to contain 'Go', got: %s", output)
	}
}

// =============================================================================
// Test: Code Block Handling
// =============================================================================

func TestCompleteCodeBlock(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "func main()") {
		t.Errorf("expected output to contain 'func main()', got: %s", output)
	}
	if !containsText(output, "Hello") {
		t.Errorf("expected output to contain 'Hello', got: %s", output)
	}
}

func TestCodeBlockWithLanguage(t *testing.T) {
	languages := []struct {
		lang string
		code string
	}{
		{"go", "package main\n\nfunc main() {}"},
		{"python", "def main():\n    print('hello')"},
		{"javascript", "function main() {\n  console.log('hello');\n}"},
	}

	for _, tc := range languages {
		t.Run(tc.lang, func(t *testing.T) {
			r, buf := createTestRenderer(t)

			r.ProcessDelta("```" + tc.lang + "\n" + tc.code + "\n```\n\n")
			r.Flush()

			output := buf.String()
			// Verify the code content is present
			if tc.lang == "go" && !containsText(output, "func main") {
				t.Errorf("expected output to contain code for %s, got: %s", tc.lang, output)
			}
			if tc.lang == "python" && !containsText(output, "def main") {
				t.Errorf("expected output to contain code for %s, got: %s", tc.lang, output)
			}
			if tc.lang == "javascript" && !containsText(output, "function main") {
				t.Errorf("expected output to contain code for %s, got: %s", tc.lang, output)
			}
		})
	}
}

func TestPartialCodeBlock(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Send partial code block (not closed)
	r.ProcessDelta("```go\nfunc main() {\n")

	// Buffer should contain content but not render yet
	if !r.IsInCodeBlock() {
		t.Error("should be tracking open code block")
	}

	// Content should still be in buffer
	bufContent := r.GetBufferContent()
	if !containsText(bufContent, "func main()") {
		t.Errorf("buffer should contain partial code, got: %s", bufContent)
	}

	// Output should be empty (waiting for code block to close)
	if buf.String() != "" {
		t.Errorf("should not output partial code block, got: %s", buf.String())
	}

	// Now close the code block
	r.ProcessDelta("}\n```\n\n")

	// Flush remaining content
	r.Flush()

	output := buf.String()
	if !containsText(output, "func main()") {
		t.Errorf("expected output to contain complete code, got: %s", output)
	}
}

func TestMultipleCodeBlocks(t *testing.T) {
	r, buf := createTestRenderer(t)

	content := `Here's Go code:

` + "```go\nfmt.Println(\"Go\")\n```" + `

And Python:

` + "```python\nprint(\"Python\")\n```" + `

`

	r.ProcessDelta(content)
	r.Flush()

	output := buf.String()
	if !containsText(output, "Go") {
		t.Errorf("expected output to contain 'Go', got: %s", output)
	}
	if !containsText(output, "Python") {
		t.Errorf("expected output to contain 'Python', got: %s", output)
	}
}

func TestCodeBlockWithTildes(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("~~~python\nprint('hello')\n~~~\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "print") {
		t.Errorf("expected output to contain 'print', got: %s", output)
	}
}

func TestNestedCodeFences(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Content with backticks inside code block (not actual nesting)
	r.ProcessDelta("```markdown\nUse `code` here\n```\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "code") {
		t.Errorf("expected output to contain 'code', got: %s", output)
	}
}

// =============================================================================
// Test: Streaming Behavior
// =============================================================================

func TestProcessDeltaCharByChar(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Stream character by character
	content := "Hello World\n\n"
	for _, c := range content {
		r.ProcessDelta(string(c))
	}
	r.Flush()

	output := buf.String()
	if !containsText(output, "Hello World") {
		t.Errorf("expected output to contain 'Hello World', got: %s", output)
	}
}

func TestProcessDeltaWordByWord(t *testing.T) {
	r, buf := createTestRenderer(t)

	words := []string{"Hello ", "World ", "from ", "Go!\n\n"}
	streamDeltas(r, words)
	r.Flush()

	output := buf.String()
	if !containsText(output, "Hello") {
		t.Errorf("expected output to contain 'Hello', got: %s", output)
	}
	if !containsText(output, "Go!") {
		t.Errorf("expected output to contain 'Go!', got: %s", output)
	}
}

func TestProcessDeltaLineByLine(t *testing.T) {
	r, buf := createTestRenderer(t)

	lines := []string{"Line 1\n", "Line 2\n", "Line 3\n", "\n"}
	streamDeltas(r, lines)
	r.Flush()

	output := buf.String()
	if !containsText(output, "Line 1") {
		t.Errorf("expected output to contain 'Line 1', got: %s", output)
	}
	if !containsText(output, "Line 3") {
		t.Errorf("expected output to contain 'Line 3', got: %s", output)
	}
}

func TestProcessDeltaMixedChunks(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Mix of different chunk sizes
	deltas := []string{
		"# ",
		"Title",
		"\n\nThis is ",
		"a paragraph with **bold** text.\n\n",
	}
	streamDeltas(r, deltas)
	r.Flush()

	output := buf.String()
	if !containsText(output, "Title") {
		t.Errorf("expected output to contain 'Title', got: %s", output)
	}
	if !containsText(output, "bold") {
		t.Errorf("expected output to contain 'bold', got: %s", output)
	}
}

func TestBufferAccumulation(t *testing.T) {
	r, _ := createTestRenderer(t)

	// Send content without ending newlines
	r.ProcessDelta("Hello")
	if r.GetBufferContent() != "Hello" {
		t.Errorf("buffer should contain 'Hello', got: %s", r.GetBufferContent())
	}

	r.ProcessDelta(" World")
	if r.GetBufferContent() != "Hello World" {
		t.Errorf("buffer should contain 'Hello World', got: %s", r.GetBufferContent())
	}
}

func TestElementBoundaryDetection(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Send a complete paragraph
	r.ProcessDelta("First paragraph.\n\n")

	// This should have been rendered
	if buf.Len() == 0 {
		t.Error("first paragraph should have been rendered after double newline")
	}

	// Send second paragraph
	r.ProcessDelta("Second paragraph.\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "First paragraph") {
		t.Errorf("expected output to contain 'First paragraph', got: %s", output)
	}
	if !containsText(output, "Second paragraph") {
		t.Errorf("expected output to contain 'Second paragraph', got: %s", output)
	}
}

// =============================================================================
// Test: Flush Behavior
// =============================================================================

func TestFlushEmptyBuffer(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Flush with empty buffer should not produce output
	r.Flush()

	if buf.Len() != 0 {
		t.Errorf("flush of empty buffer should produce no output, got: %s", buf.String())
	}
}

func TestFlushPartialContent(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Send content without paragraph break
	r.ProcessDelta("Partial content without newline")

	// Should be buffered, not rendered
	beforeFlush := buf.String()

	r.Flush()

	// After flush, content should be rendered
	afterFlush := buf.String()
	if len(afterFlush) <= len(beforeFlush) {
		t.Error("flush should render remaining content")
	}
	if !containsText(afterFlush, "Partial content") {
		t.Errorf("expected flushed output to contain content, got: %s", afterFlush)
	}
}

func TestFlushCompleteContent(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("Complete paragraph.\n\n")
	beforeFlush := buf.String()

	r.Flush()
	afterFlush := buf.String()

	// Content should have been rendered at paragraph boundary
	// Flush might add a bit more but main content should be present
	if !containsText(beforeFlush, "Complete paragraph") && !containsText(afterFlush, "Complete paragraph") {
		t.Errorf("content should be rendered, got before: %s, after: %s", beforeFlush, afterFlush)
	}
}

func TestFlushIncompleteCodeBlock(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Send incomplete code block
	r.ProcessDelta("```go\nfunc incomplete() {\n")

	// Flush should render it anyway
	r.Flush()

	output := buf.String()
	if !containsText(output, "func incomplete") {
		t.Errorf("flush should render incomplete code block, got: %s", output)
	}
}

func TestMultipleFlushCalls(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("Some content\n\n")
	r.Flush()
	firstFlush := buf.String()

	// Second flush should not add duplicate content
	r.Flush()
	secondFlush := buf.String()

	if firstFlush != secondFlush {
		t.Errorf("multiple flushes should not duplicate content, first: %s, second: %s", firstFlush, secondFlush)
	}
}

// =============================================================================
// Test: Edge Cases
// =============================================================================

func TestPlainTextNoMarkdown(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("Just plain text without any markdown formatting.\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "Just plain text") {
		t.Errorf("plain text should render correctly, got: %s", output)
	}
}

func TestMixedContentCodeAndProse(t *testing.T) {
	r, buf := createTestRenderer(t)

	content := `Here's an explanation:

The function works as follows:

` + "```go\nfunc Add(a, b int) int {\n    return a + b\n}\n```" + `

This adds two numbers together.

`
	r.ProcessDelta(content)
	r.Flush()

	output := buf.String()
	if !containsText(output, "explanation") {
		t.Errorf("expected prose content, got: %s", output)
	}
	if !containsText(output, "func Add") {
		t.Errorf("expected code content, got: %s", output)
	}
	if !containsText(output, "adds two numbers") {
		t.Errorf("expected trailing prose, got: %s", output)
	}
}

func TestEmptyDeltas(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Send empty deltas
	r.ProcessDelta("")
	r.ProcessDelta("")
	r.ProcessDelta("Hello\n\n")
	r.ProcessDelta("")
	r.Flush()

	output := buf.String()
	if !containsText(output, "Hello") {
		t.Errorf("should handle empty deltas gracefully, got: %s", output)
	}
}

func TestVeryLongLines(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Create a very long line
	longWord := strings.Repeat("a", 200)
	r.ProcessDelta(longWord + "\n\n")
	r.Flush()

	output := buf.String()
	// Glamour should wrap long lines, but content should be present
	if !containsText(output, "aaa") {
		t.Errorf("long line should be rendered (possibly wrapped), got: %s", output)
	}
}

func TestSpecialCharacters(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("Special chars: <>&\"' and unicode: 日本語 🚀\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "Special chars") {
		t.Errorf("special characters should render, got: %s", output)
	}
}

func TestMarkdownEscaping(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("Use \\*escaped\\* asterisks.\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "escaped") {
		t.Errorf("escaped markdown should render, got: %s", output)
	}
}

// =============================================================================
// Test: State Management
// =============================================================================

func TestResetClearsState(t *testing.T) {
	r, _ := createTestRenderer(t)

	// Add content and create state
	r.ProcessDelta("```go\nsome code\n")

	if !r.IsInCodeBlock() {
		t.Error("should be in code block before reset")
	}
	if r.GetBufferContent() == "" {
		t.Error("buffer should have content before reset")
	}

	r.Reset()

	if r.IsInCodeBlock() {
		t.Error("should not be in code block after reset")
	}
	if r.GetBufferContent() != "" {
		t.Errorf("buffer should be empty after reset, got: %s", r.GetBufferContent())
	}
}

func TestStateTracking(t *testing.T) {
	r, _ := createTestRenderer(t)

	// Not in code block initially
	if r.IsInCodeBlock() {
		t.Error("should not be in code block initially")
	}

	// Enter code block
	r.ProcessDelta("```go\n")
	if !r.IsInCodeBlock() {
		t.Error("should be in code block after opening fence")
	}

	// Still in code block
	r.ProcessDelta("func main() {\n")
	if !r.IsInCodeBlock() {
		t.Error("should still be in code block")
	}

	// Exit code block
	r.ProcessDelta("}\n```\n")
	if r.IsInCodeBlock() {
		t.Error("should not be in code block after closing fence")
	}
}

func TestMultipleMessagesSequentially(t *testing.T) {
	r, buf := createTestRenderer(t)

	// First message
	r.ProcessDelta("First message content.\n\n")
	r.Flush()

	firstOutput := buf.String()
	if !containsText(firstOutput, "First message") {
		t.Errorf("first message should render, got: %s", firstOutput)
	}

	// Reset for second message
	r.Reset()

	// Second message
	r.ProcessDelta("Second message content.\n\n")
	r.Flush()

	// Buffer now has both (since we didn't create new buffer)
	totalOutput := buf.String()
	if !containsText(totalOutput, "First message") {
		t.Errorf("should still have first message, got: %s", totalOutput)
	}
	if !containsText(totalOutput, "Second message") {
		t.Errorf("should have second message, got: %s", totalOutput)
	}
}

func TestRendererReuse(t *testing.T) {
	r, buf := createTestRenderer(t)

	// First use
	r.ProcessDelta("First\n\n")
	r.Flush()

	// Reset and reuse
	r.Reset()
	r.ProcessDelta("Second\n\n")
	r.Flush()

	// Reset and reuse again
	r.Reset()
	r.ProcessDelta("Third\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "First") {
		t.Errorf("should contain First, got: %s", output)
	}
	if !containsText(output, "Second") {
		t.Errorf("should contain Second, got: %s", output)
	}
	if !containsText(output, "Third") {
		t.Errorf("should contain Third, got: %s", output)
	}
}

// =============================================================================
// Test: Integration with Different Content Types
// =============================================================================

func TestComplexMarkdownDocument(t *testing.T) {
	r, buf := createTestRenderer(t)

	content := `# Main Title

This is an introduction paragraph with **bold** and *italic* text.

## Code Example

Here's a Go function:

` + "```go\n// Add adds two integers\nfunc Add(a, b int) int {\n    return a + b\n}\n```" + `

## Features

- Feature one
- Feature two
- Feature three

### Nested Section

1. First step
2. Second step
3. Third step

> This is a blockquote

` + "`inline code`" + ` is also supported.

`
	r.ProcessDelta(content)
	r.Flush()

	output := buf.String()

	// Check various elements are present
	checks := []string{
		"Main Title",
		"introduction",
		"Code Example",
		"func Add",
		"Features",
		"Feature one",
		"First step",
		"blockquote",
		"inline code",
	}

	for _, check := range checks {
		if !containsText(output, check) {
			t.Errorf("expected output to contain '%s', got: %s", check, output)
		}
	}
}

func TestStreamingCodeBlockCharByChar(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Stream a code block character by character
	codeBlock := "```go\nfmt.Println(\"Hello\")\n```\n\n"
	for _, c := range codeBlock {
		r.ProcessDelta(string(c))
	}
	r.Flush()

	output := buf.String()
	if !containsText(output, "fmt.Println") {
		t.Errorf("streamed code block should render correctly, got: %s", output)
	}
}

// =============================================================================
// Test: Error Handling / Robustness
// =============================================================================

func TestMalformedMarkdown(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Malformed markdown that might confuse parsers
	malformed := `**unclosed bold

# Header without content

- List item
  - Nested without proper indent
- Another item

[broken link(no closing bracket

` + "```" + `
code without language or closing fence
`
	r.ProcessDelta(malformed)
	r.Flush()

	// Should not panic, should produce some output
	output := buf.String()
	if output == "" {
		t.Error("malformed markdown should still produce output")
	}
}

func TestRapidDeltaProcessing(t *testing.T) {
	r, buf := createTestRenderer(t)

	// Simulate rapid delta processing
	for i := 0; i < 100; i++ {
		r.ProcessDelta("word ")
	}
	r.ProcessDelta("\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "word") {
		t.Errorf("rapid processing should work, got: %s", output)
	}
}

func TestUnicodeContent(t *testing.T) {
	r, buf := createTestRenderer(t)

	r.ProcessDelta("## Unicode Test 🎉\n\n日本語テスト\n\nEmojis: 👍 ✅ 🚀\n\n")
	r.Flush()

	output := buf.String()
	if !containsText(output, "Unicode Test") {
		t.Errorf("unicode should render, got: %s", output)
	}
}
