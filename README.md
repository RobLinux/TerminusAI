# TerminusAI

CLI AI agent that plans and runs console commands with your approval.

## Features

- **Smart Command Planning**: Uses AI to break down tasks into executable commands
- **Multiple LLM Providers**: Support for OpenAI, Anthropic (Claude), and GitHub Models
- **Interactive Agent Mode**: Iterative task execution with file inspection and step-by-step approvals
- **Security-Focused**: All commands require user approval with persistent policy rules
- **Cross-Platform**: Works on Windows, macOS, and Linux

## Installation

### From Source

```bash
# Clone and build
git clone <repository-url>
cd terminusai
make build

# Or install directly
make install
```

### Prerequisites

- Go 1.21 or later
- One of the supported AI providers:
  - OpenAI API key
  - Anthropic API key  
  - GitHub Token with Models access

## Quick Start

1. **Setup**: Configure your AI provider
   ```bash
   ./terminusai setup
   ```

2. **Run a task**: Ask the AI to plan and execute commands
   ```bash
   ./terminusai run "create a docker image from this directory"
   ```

3. **Agent mode**: Let the AI inspect files and work iteratively
   ```bash
   ./terminusai agent "build this project into an executable"
   ```

## Commands

### `terminusai run <task...>`
Ask the agent to perform a task by planning commands to run.

**Options:**
- `--provider <name>`: LLM provider (openai|anthropic|github)
- `--model <id>`: Model ID override
- `--setup`: Run setup wizard before executing
- `--verbose`: Enable verbose logging
- `--debug`: Enable maximum debug logging
- `--dry-run`: Only show the plan, do not execute

**Example:**
```bash
terminusai run "install dependencies and start the development server"
```

### `terminusai agent <task...>`
Iterative agent mode where the AI inspects files and runs commands step-by-step.

**Options:**
- `--provider <name>`: LLM provider (openai|anthropic|github)
- `--model <id>`: Model ID override
- `--plan-first`: Generate a plan first, then execute with approvals
- `--dry-run`: With --plan-first, show plan only and exit
- `--verbose`: Enable verbose logging
- `--debug`: Enable maximum debug logging

**Example:**
```bash
terminusai agent "analyze this codebase and create comprehensive documentation"
```

### `terminusai setup`
Run the interactive setup wizard to configure providers and credentials.

### `terminusai model`
Set or override the preferred model.

**Options:**
- `--provider <name>`: Provider to set model for (defaults to current)
- `--model <id>`: Model ID to set (if omitted, you will be prompted)

## Configuration

Configuration is stored in `~/.terminusai/config.json` and includes:

- **Provider settings**: Default provider and model
- **API credentials**: Securely stored API keys and tokens
- **Command policies**: Persistent approval rules in `~/.terminusai/policy.json`

## AI Providers

### OpenAI
- **Models**: gpt-4o, gpt-4o-mini, o4-mini
- **Setup**: Requires OPENAI_API_KEY

### Anthropic (Claude)
- **Models**: claude-3-5-sonnet-latest, claude-3-5-haiku-latest
- **Setup**: Requires ANTHROPIC_API_KEY

### GitHub Models
- **Models**: gpt-4o, gpt-4o-mini (via GitHub)
- **Setup**: Requires GITHUB_TOKEN with Models/Copilot access

## Security

TerminusAI prioritizes security:

- **Command Approval**: Every command requires explicit user approval
- **Policy Rules**: Create persistent rules to always/never allow specific commands
- **No Auto-execution**: Commands are never run without permission
- **Secure Storage**: API keys stored locally in config files

## Environment Variables

- `TERMINUS_AI_VERBOSE=1`: Enable verbose logging
- `TERMINUS_AI_DEBUG=1`: Enable debug logging  
- `TERMINUS_AI_TEMPERATURE`: Set LLM temperature (0.0-1.0)
- `TERMINUS_AI_DEFAULT_MODEL`: Override default model
- `TERMINUS_AI_DEFAULT_PROVIDER`: Override default provider

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Build for all platforms
make build-all

# Clean build artifacts  
make clean
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license information here]