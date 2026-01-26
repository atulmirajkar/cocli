# CoCli

An interactive Go CLI tool for quick question answer mode in the cli. This is build using GitHub Copilot SDK. This tool allows you to have conversations with different AI models, track token usage, and switch between models seamlessly.

## Features

- **Interactive CLI** - Chat with AI models in real-time
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
# With or without quotes - both work!
cocli What is the capital of France
cocli "What is the capital of France"
cocli Write a function that calculates fibonacci numbers
```

All arguments are concatenated with spaces to form the prompt. The tool will process your prompt and then enter interactive mode for follow-up questions.

The tool will start and display a prompt with the current model and token information:

```
[gpt-4.1 | 4000/4000 tokens] >
```

### Available Commands

#### Chat with AI

Simply type your prompt and press Enter (or pass it as arguments):

```
[gpt-4.1 | 4000/4000 tokens] > What is the capital of France?
```

The response will be streamed in real-time to your terminal.

#### List Available Models

Type `/models` or `/list` to see all available models:

```
[gpt-4.1 | 3500/4000 tokens] > /models

Fetching available models from server...
Available models:
* 1. GPT-4.1 (ID: gpt-4.1)
  2. GPT-5 (ID: gpt-5)
  3. Claude Haiku (ID: claude-haiku-4.5)
  4. Claude Sonnet (ID: claude-sonnet-4.5)
  ...
```

#### Switch Models

When viewing the model list, enter the number corresponding to the model you want to use:

```
Enter model number (current: gpt-4.1, press Enter to skip): 2
Switched to: gpt-5
```

A new session will be created with the selected model, and token counters will reset.

#### Exit the Tool

Press `Ctrl+C` to exit gracefully:

```
[gpt-4.1 | 2500/4000 tokens] > ^C
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
│   └── session.go               # Session manager - encapsulates SDK client,
│                                # session creation, and event handling
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
[model | tokens_remaining/token_limit] >
```

- **tokens_remaining** = token_limit - current_tokens_used
- **token_limit** = Maximum tokens available for the session
- Resets to 0/0 when switching models (each model has its own limit)

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

Use the included build script to create cross-platform binaries for all platforms:

```bash
./scripts/build.sh
```

This will automatically build binaries for:

- macOS (Apple Silicon) - `cocli-darwin-arm64`
- macOS (Intel) - `cocli-darwin-amd64`
- Linux (x86_64) - `cocli-linux-amd64`
- Linux (ARM64) - `cocli-linux-arm64`

The binaries will be created in the `dist/` directory.

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

1. Make your changes and test locally with `go run main.go`
2. Commit your changes and create a git tag
3. Run `./scripts/build.sh` to build binaries for all platforms
4. Copy binaries from `dist/` to `releases/`
5. Create a GitHub release with the binaries attached
6. Update the download URLs in the README with the new release version

## License

[Add your license information here]

## Contributing

[Add contribution guidelines here]
