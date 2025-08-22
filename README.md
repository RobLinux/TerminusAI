# 🤖 TerminusAI

**Your intelligent CLI companion that transforms natural language into executable commands**

TerminusAI is a powerful CLI tool that uses AI to understand your tasks and generate the right commands to accomplish them. With built-in safety features and multi-provider support, it's like having an expert assistant for your terminal.

## ✨ Key Features

🧠 **Smart Command Planning** - AI breaks down complex tasks into step-by-step commands  
🔌 **Multi-Provider Support** - Works with OpenAI, Anthropic Claude, and GitHub Models  
🔍 **Interactive Agent Mode** - Inspects files and executes tasks iteratively  
🛡️ **Security First** - Every command requires your approval with persistent policies  
🌍 **Cross-Platform** - Runs seamlessly on Windows, macOS, and Linux

## 🚀 Quick Setup

### Install from Source
```bash
git clone <repository-url>
cd terminusai
make install
```

### Prerequisites
- **Go 1.21+**
- **API Key** from one of:
  - OpenAI (GPT-4o, o4-mini)
  - Anthropic (Claude 3.5 Sonnet/Haiku)
  - GitHub (Models access)

## 🏃 Get Started in 30 Seconds

```bash
# 1. Configure your AI provider
./terminusai setup

# 2. Run any task with natural language
./terminusai run "create a docker image from this directory"

# 3. Use agent mode for complex, multi-step tasks
./terminusai agent "build this project into an executable"
```

## 📖 Usage

### Core Commands

| Command | Description | Example |
|---------|-------------|---------|
| `terminusai run <task>` | Execute single tasks with AI planning | `terminusai run "deploy to staging"` |
| `terminusai agent <task>` | Interactive mode for complex tasks | `terminusai agent "refactor this codebase"` |
| `terminusai setup` | Configure AI providers & credentials | `terminusai setup` |
| `terminusai model` | Change AI model settings | `terminusai model --provider openai` |

### Common Flags
- `--provider` - Choose AI provider (openai/anthropic/github)
- `--dry-run` - Show plan without executing
- `--verbose` - Detailed logging

## ⚙️ Configuration

Settings stored in `~/.terminusai/`:
- `config.json` - Provider settings and API credentials
- `policy.json` - Command approval rules

### Supported AI Providers

| Provider | Models | Required Key |
|----------|--------|--------------|
| **OpenAI** | GPT-4o, GPT-4o-mini, o4-mini | `OPENAI_API_KEY` |
| **Anthropic** | Claude 3.5 Sonnet/Haiku | `ANTHROPIC_API_KEY` |
| **GitHub** | GPT-4o (via GitHub) | `GITHUB_TOKEN` |

## 🔐 Security

TerminusAI puts safety first:
- ✅ **Every command needs your approval**
- 📋 **Persistent approval policies** 
- 🚫 **No auto-execution**
- 🔒 **Local credential storage**

## 🔧 Environment Variables

| Variable | Description |
|----------|-------------|
| `TERMINUS_AI_VERBOSE=1` | Enable verbose logging |
| `TERMINUS_AI_DEBUG=1` | Enable debug logging |
| `TERMINUS_AI_TEMPERATURE` | Set LLM temperature (0.0-1.0) |
| `TERMINUS_AI_DEFAULT_MODEL` | Override default model |
| `TERMINUS_AI_DEFAULT_PROVIDER` | Override default provider |

## 🛠️ Development

```bash
make deps      # Install dependencies
make test      # Run tests  
make build-all # Build for all platforms
make clean     # Clean artifacts
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch  
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

---

**Ready to supercharge your terminal experience?** Get started with TerminusAI today! 🚀