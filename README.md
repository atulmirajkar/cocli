# CoCli

An interactive Go CLI tool for quick question answer mode in the cli. This is build using GitHub Copilot SDK. This tool allows you to have conversations with different AI models, track token usage, and switch between models seamlessly.

## Features

- **Interactive CLI** - Chat with AI models in real-time
- **Markdown Rendering** - Beautiful syntax-highlighted output with dark theme
- **Dynamic Model Selection** - List and switch between available models on the fly
- **Token Tracking** - Monitor token usage during conversations
- **Model Persistence** - Maintains the current model throughout the session
- **Graceful Shutdown** - Press Ctrl+C to exit cleanly

## Prerequisites

- GitHub Copilot Agent running locally
- GitHub Copilot authentication configured

## Installation

### Quick Start with Pre-built Binaries

#### macOS

1. **Install GitHub Copilot CLI**:

    ```bash
    brew install github-copilot-cli
    ```

2. **Download the binary**:

    ```bash
    # For Apple Silicon (M1/M2/M3)
    curl -L https://raw.githubusercontent.com/atulmirajkar/cocli/main/releases/cocli-darwin-arm64 -o cocli
    chmod +x cocli

    # For Intel Mac
    curl -L https://raw.githubusercontent.com/atulmirajkar/cocli/main/releases/cocli-darwin-amd64 -o cocli
    chmod +x cocli

    ./cocli
    ```

#### Linux

1. **Install GitHub Copilot CLI**:

    Follow instructions at https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli

2. **Download the binary**:

    ```bash
    # For x86_64
    curl -L https://raw.githubusercontent.com/atulmirajkar/cocli/main/releases/cocli-linux-amd64 -o cocli
    chmod +x cocli

    # For ARM64 (Raspberry Pi, etc.)
    curl -L https://raw.githubusercontent.com/atulmirajkar/cocli/main/releases/cocli-linux-arm64 -o cocli
    chmod +x cocli

    ./cocli
    ```

### Build from Source

If you prefer to build from source or the pre-built binary for your platform is not available:

#### Prerequisites for Building

- Go 1.23.0 or higher

#### macOS

1. **Install Go and Copilot CLI**:

    ```bash
    brew install copilot-cli
    ```

2. **Clone the repository**:

    ```bash
    git clone git@github.com:atulmirajkar/cocli.git
    cd cocli
    ```

3. **Install dependencies and build**:
    ```bash
    go mod download
    go build -o cocli
    ./cocli
    ```

#### Linux

1. **Install Go and Copilot CLI**:

    https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli`

2. **Clone the repository**:

    ```bash
    git clone git@github.com:atulmirajkar/cocli.git
    cd cocli
    ```

3. **Install dependencies and build**:
    ```bash
    go mod download
    go build -o cocli
    ./cocli
    ```

## Usage

### Starting the Tool

You can start the tool in two ways:

**Interactive mode** (no arguments):

```bash
cocli
```

**With an initial prompt** (one or more arguments):

```bash
# Simple queries
cocli What is the capital of France
cocli Write a function that calculates fibonacci numbers
```

All arguments are concatenated with spaces to form the prompt. The tool will process your prompt and then enter interactive mode for follow-up questions.

#### Note on Special Characters

When using command-line arguments, shell special characters like `?`, `*`, `&`, `|`, `$`, and backticks may be interpreted by your shell. **Always use quotes for queries with special characters:**

```bash
# ✓ Correct - Use quotes for safety
cocli "Does this pattern match: *.txt?"
cocli "Check if x > 5 && y < 10"
cocli "What's the regex for emails?"

# ✗ Avoid without quotes - shell may expand these
cocli Does this pattern match: *.txt?
cocli Check if x > 5 && y < 10
```

**Best practice**: Quote your entire prompt if it contains any special characters.

The tool will start and display a prompt with the current model and token information:

```
[Claude Sonnet 4.5 | 1.00x | 4000/4000 tokens] >
```

### Available Commands

#### Chat with AI

Simply type your prompt and press Enter (or pass it as arguments):

```
[Claude Sonnet 4.5 | 1.00x | 3500/4000 tokens] > What is the capital of France?
```

The response will be streamed in real-time to your terminal with markdown formatting and syntax highlighting.

#### List Available Models

Type `/models` or `/list` to see all available models:

```
[Claude Sonnet 4.5 | 1.00x | 3500/4000 tokens] > /models

Fetching available models from server...
Available models:
  1. Claude Haiku 4.5 (ID: claude-haiku-4.5) (0.33x)
* 2. Claude Sonnet 4.5 (ID: claude-sonnet-4.5) (1.00x)
  3. GPT-4.1 (ID: gpt-4.1) (2.00x)
  ...
```

#### Switch Models

When viewing the model list, enter the number corresponding to the model you want to use:

```
Enter model number (current: Claude Sonnet 4.5, press Enter to skip): 1
Switched to: Claude Haiku 4.5 (0.33x)
```

A new session will be created with the selected model, and token counters will reset.

#### Exit the Tool

Press `Ctrl+C` to exit gracefully:

```
[Claude Sonnet 4.5 | 1.00x | 2500/4000 tokens] > ^C
Bye
```

## Project Structure

```
cocli/
├── main.go                      # CLI entry point and interactive loop
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency checksums
├── .gitignore                   # Git ignore rules for Go projects
│
├── session/
│   ├── session.go               # Session manager - encapsulates SDK client,
│   │                            # session creation, and event handling
│   ├── session_test.go          # Unit tests for session manager
│   ├── renderer.go              # Streaming markdown renderer with syntax highlighting
│   └── renderer_test.go         # Unit tests for markdown renderer
│
├── scripts/
│   └── build.sh                 # Cross-platform build script for all platforms
│
├── releases/
│   ├── README.md                # Release documentation
│   ├── cocli-darwin-arm64       # macOS Apple Silicon binary
│   ├── cocli-darwin-amd64       # macOS Intel binary
│   ├── cocli-linux-amd64        # Linux x86_64 binary
│   └── cocli-linux-arm64        # Linux ARM64 binary
│
├── TESTING.md                   # Manual testing checklist for markdown rendering
└── README.md                    # This file
```

### Directory Overview

- **root** - Main Go source files and configuration
- **session/** - Package for SDK client and session management
- **scripts/** - Build and utility scripts
- **releases/** - Pre-built binaries for distribution

## How It Works

### Session Manager

The `session` package encapsulates all Copilot SDK interactions:

- **Client Management** - Handles connection to the Copilot Agent
- **Session Creation** - Creates new sessions with specified models
- **Event Handling** - Processes streaming responses and tracks token usage
- **Model Listing** - Queries available models from the server

### Main CLI Loop

The main program:

1. Initializes the session manager
2. Sets up signal handling for graceful shutdown
3. Maintains an interactive loop for user input
4. Routes commands (`/models`) and prompts appropriately
5. Tracks the current model and token usage

## Token Tracking

Token usage is displayed in the prompt format:

```
[model | multiplier | tokens_remaining/token_limit] >
```

- **tokens_remaining** = token_limit - current_tokens_used
- **token_limit** = Maximum tokens available for the session
- Resets to 0/0 when switching models (each model has its own limit)

## Markdown Rendering

The CLI automatically renders markdown responses with beautiful formatting:

- **Model-Level Formatting** - The AI model is instructed to always output markdown for consistent, high-quality responses
- **Dark Theme** - Optimized for terminal readability
- **Syntax Highlighting** - Code blocks with language-specific coloring (Go, Python, JavaScript, etc.)
- **Real-time Streaming** - Markdown is rendered incrementally as responses arrive
- **Formatted Elements** - Headers, lists, bold, italic, inline code, and links are properly styled

The system uses a dual approach: a system message instructs the model to format responses in markdown, while the streaming renderer ensures beautiful display. No configuration needed—it works automatically!

## Troubleshooting

### "SDK protocol version mismatch" Error

This occurs when the Copilot Agent doesn't match the SDK's expected protocol version. Solutions:

1. Update GitHub Copilot CLI: `brew upgrade github-copilot-cli` (macOS)
2. Ensure the Copilot Agent is running and accessible
3. Check your Copilot authentication: `copilot --version`

### "No models available" Error

- Ensure the Copilot Agent is running
- Check your internet connection
- Verify Copilot authentication is configured

### Connection Refused

- Make sure the Copilot Agent is running locally
- Check that no firewall is blocking the connection

## Development

### Building Binaries for Distribution

**Recommended:** Use the release script for a complete build and validation process:

```bash
# Auto-generates version from current date
./scripts/release.sh

# With custom version
./scripts/release.sh "1.2.3"
```

This will automatically:

- Run all tests
- Build cross-platform binaries for all platforms
- Update the `releases/` directory
- Validate binary architectures
- Provide git commands for release completion

**Alternative:** Use the build script directly (creates binaries in `dist/` directory):

```bash
./scripts/build.sh
```

This will build binaries for:

- macOS (Apple Silicon) - `cocli-darwin-arm64`
- macOS (Intel) - `cocli-darwin-amd64`
- Linux (x86_64) - `cocli-linux-amd64`
- Linux (ARM64) - `cocli-linux-arm64`

If you prefer to build manually or for a specific platform:

```bash
# Build for macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o cocli-darwin-arm64

# Build for macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o cocli-darwin-amd64

# Build for Linux (x86_64)
GOOS=linux GOARCH=amd64 go build -o cocli-linux-amd64

# Build for Linux (ARM64)
GOOS=linux GOARCH=arm64 go build -o cocli-linux-arm64
```

### Local Development

To modify the tool:

1. Update `session/session.go` for SDK or session management changes
2. Update `main.go` for CLI behavior changes
3. Test with: `go run main.go`
4. Build for your platform: `go build -o cocli`

### Release Process

#### Automated Release (Recommended)

Use the automated release script for binary building and validation:

```bash
# Basic usage - auto-generates version from current date
./scripts/release.sh

# With custom version
./scripts/release.sh "1.2.3"
```

**What the script does automatically:**

1. Runs all tests to ensure code quality
2. Verifies the application builds locally
3. Creates cross-platform binaries for all supported platforms
4. Updates the `releases/` directory with new binaries
5. Validates binary architectures and permissions
6. Cleans up temporary build files
7. Provides step-by-step git commands for manual execution

**After running the script, complete the release manually:**

The script will output git commands to run. Follow the "Manual steps remaining" from the script output.

## License

[Add your license information here]

## Contributing

[Add contribution guidelines here]
