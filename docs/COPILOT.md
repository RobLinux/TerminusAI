# GitHub Copilot Integration

TerminusAI now supports GitHub Copilot's completion API for code generation and assistance.

## Setup

### 1. Authenticate with GitHub Copilot

```bash
terminusai copilot auth
```

This will:
- Start the GitHub Device Flow authentication
- Copy the device code to your clipboard automatically
- Open your browser to complete authentication
- Save the access token for future use

### 2. Use Copilot Provider

After authentication, you can use the Copilot provider:

```bash
# Use Copilot for code completion
terminusai agent "write a function to calculate fibonacci numbers" --provider copilot-api

# Use Copilot for code analysis
terminusai agent "explain this code and suggest improvements" --provider copilot-api
```

## How it Works

The Copilot integration:

1. **Authentication**: Uses GitHub's Device Flow OAuth to authenticate with Copilot
2. **Token Management**: Automatically handles session token refresh
3. **Completion API**: Uses the real GitHub Copilot completion endpoint
4. **Streaming**: Supports streaming responses for real-time feedback

## API Details

The implementation follows the GitHub Copilot API specification:

- **Endpoint**: `https://copilot-proxy.githubusercontent.com/v1/engines/copilot-codex/completions`
- **Authentication**: Bearer token from GitHub Copilot session
- **Streaming**: JSON streaming response format
- **Language Support**: Supports language-specific completions

## Configuration

The Copilot provider stores authentication tokens in:
- Access Token: `~/.copilot_token`
- Session Token: Managed automatically in memory

## Troubleshooting

### Authentication Issues

If authentication fails, try:
1. Ensure you have access to GitHub Copilot
2. Check your GitHub account has Copilot enabled
3. Re-run `terminusai copilot auth`

### API Errors

Common issues:
- **401 Unauthorized**: Token expired, re-authenticate
- **403 Forbidden**: No Copilot access or quota exceeded
- **Connection errors**: Network or proxy issues

## Examples

### Code Generation
```bash
terminusai agent "create a REST API server in Go" --provider copilot-api
```

### Code Review
```bash
terminusai agent "review this code for security issues" --provider copilot-api
```

### Debugging
```bash
terminusai agent "debug this error and suggest fixes" --provider copilot-api
```
