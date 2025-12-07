# Katichai

**Context-aware, AI-assisted code review CLI tool**

Katichai prevents unnecessary AI-generated code, detects duplicated logic, enforces architectural patterns, and ensures high-quality engineering standards.

## Features

- 🧠 **Semantic Code Understanding** - Builds deep context of your codebase using embeddings and static analysis
- 🔍 **AI Code Detection** - Identifies unnecessary AI-generated boilerplate and verbose code
- 🔄 **Duplicate Detection** - Finds exact and semantic code duplication across your repository
- 🏗️ **Architecture Enforcement** - Detects frameworks and enforces their conventions
- 🌐 **Multi-Language Support** - Works with Go, Java, Python, JavaScript, TypeScript, and more
- 🚀 **Offline-First** - Runs locally with minimal LLM usage

## Installation

```bash
# Coming soon
go install github.com/katichai/katich@latest
```

## Quick Start

```bash
# Build codebase context
katich context build

# Review latest commit
katich review latest

# Review a specific diff range
katich review diff HEAD~3..HEAD

# Review in CI mode
katich review --ci
```

## Supported Frameworks

- **Java**: Spring Boot
- **JavaScript/TypeScript**: Express, Next.js, React
- **Python**: FastAPI
- **Go**: Gin

## How It Works

1. **Context Building**: Scans your repository, detects frameworks, parses ASTs, and generates embeddings
2. **Diff Analysis**: Extracts changes from Git and analyzes modified functions
3. **Similarity Search**: Compares new code against existing codebase using FAISS
4. **AI Detection**: Uses heuristics and small LLM classifiers to detect AI-generated patterns
5. **Review Synthesis**: Combines static analysis with LLM reasoning for high-signal reviews

## Commands

### Context Commands
- `katich context build` - Build codebase context and embeddings
- `katich context show` - Display current context information
- `katich context clear` - Clear cached context

### Review Commands
- `katich review latest` - Review the latest commit
- `katich review diff <range>` - Review a specific commit range
- `katich review file <path>` - Review a specific file
- `katich review --ci` - Run in CI mode (exits with error code on issues)

### Utility Commands
- `katich doctor` - Check system requirements and configuration
- `katich version` - Display version information

## Configuration

Create a `.katich/config.yaml` file:

```yaml
llm:
  provider: openai  # openai, anthropic, or local
  api_key: your-api-key
  model: gpt-4

embeddings:
  model: jina-code-v2  # jina-code-v2, bge-code, nomic-embed, snowflake-arctic

analysis:
  max_function_length: 50
  complexity_threshold: 10
  similarity_threshold: 0.85
```

## Development Status

🚧 **Currently in active development** - See [tasks.md](tasks.md) for progress

## License

MIT

## Contributing

Contributions welcome! Please read our contributing guidelines first.

---

Built with ❤️ by the Katichai team
