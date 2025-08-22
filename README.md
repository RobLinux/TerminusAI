# 🤖 TerminusAI

**Your intelligent CLI companion that transforms natural language into executable commands**

TerminusAI is a powerful CLI tool that uses AI to understand your tasks and generate the right commands to accomplish them. With built-in safety features and multi-provider support, it's like having an expert assistant for your terminal.

## ✨ Key Features

🧠 **Smart Command Understanding** - AI interprets natural language and executes the right commands  
🔌 **Multi-Provider Support** - Works with OpenAI, Anthropic Claude, and GitHub Copilot  
🔍 **Interactive Agent Mode** - Inspects files and executes tasks iteratively  
🛡️ **Security First** - Every command requires your approval with persistent policies  
🌍 **Cross-Platform** - Runs seamlessly on Windows, macOS, and Linux  
⚡ **Simple Interface** - No subcommands needed, just ask what you want

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
  - GitHub (Copilot access)

## 🏃 Get Started in 30 Seconds

```bash
# 1. Configure your AI provider
./terminusai setup

# 2. Ask anything in natural language
./terminusai "1+1=?"
./terminusai "create a docker image from this directory"
./terminusai "build this project into an executable"
```

## 📖 Usage

### Simple and Direct

Just ask TerminusAI what you want to do:

```bash
# Math and general questions
terminusai "what is 2+2?"
terminusai "explain what Docker is"

# File operations
terminusai "list all Python files in this directory"
terminusai "create a README file for this project"

# Development tasks
terminusai "install dependencies and run tests"
terminusai "build this Go project"
terminusai "format all code files"

# System administration
terminusai "check disk usage"
terminusai "find large files in this directory"
```

### Utility Commands

| Command | Description | Example |
|---------|-------------|---------|
| `terminusai setup` | Configure AI providers & credentials | `terminusai setup` |
| `terminusai model` | Change AI model settings | `terminusai model --provider openai` |
| `terminusai config` | View current configuration | `terminusai config` |

### Common Flags
- `--provider` - Choose AI provider (openai/anthropic/copilot)
- `--verbose` - Detailed logging
- `--debug` - Maximum debug output

## ⚙️ Configuration

Settings stored in `~/.terminusai/`:
- `config.json` - Provider settings and API credentials
- `policy.json` - Command approval rules

### Supported AI Providers

| Provider | Models | Required Key |
|----------|--------|--------------|
| **OpenAI** | GPT-4o, GPT-4o-mini, o4-mini | `OPENAI_API_KEY` |
| **Anthropic** | Claude 3.5 Sonnet/Haiku | `ANTHROPIC_API_KEY` |
| **GitHub** | Copilot models | `GITHUB_TOKEN` |

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