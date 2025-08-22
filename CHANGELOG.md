# Changelog

## [1.0.1] - 2025-08-22

### Added
- Comprehensive tokenizer system with per-provider token management
- Enhanced agent output display with interactive command result tracking
- Massive expansion of agent capabilities and action system
- Write file action support

### Changed
- Simplified CLI interface and restructured agent architecture
- Consolidated agent architecture and simplified provider system
- Replaced GitHub provider with Copilot provider and improved shell execution
- Version bump to 1.0.1

### Removed
- Removed redundant Copilot provider components

### Fixed
- Shell execution improvements
- Provider system optimizations

## [1.0.0] - 2025-08-22

### Added
- Initial release of TerminusAI
- Agent-based AI assistant with multiple provider support
- Support for OpenAI, Anthropic, and GitHub Copilot providers
- Interactive command execution and file operations
- Configuration management system
- Policy-based security controls

### Development History
- Initial commit (2c013ad)
- Add missing files and update existing ones (55a621e)
- Refactor agent implementation: removed executor, reorganized agent code into separate files for better maintainability, deleted unused demo files, and updated command structure (2921add)
- Reduce output line limit from 5000 to 500 and add more directory exclusions (583eaca)
- Summarize changes (7f1938e)
- Refactor: Remove Copilot provider and update related components (ae4bfeb)
- refactor: Replace GitHub provider with Copilot provider and fix shell execution (14b3f59)
- feat: prepare for v1.0.0 release (5460cac)
- feat: add write_file action support and remove copilot command (c9bba17)
- refactor: consolidate agent architecture and simplify provider system (60edf13)
- refactor: simplify CLI interface and restructure agent architecture (67a7aaa)
- feat: Massive expansion of agent capabilities and action system (1ed03da)
- feat: enhance agent output display with interactive command result tracking (09333ac)
- feat: Implement comprehensive tokenizer system with per-provider token management (3c1c05c)