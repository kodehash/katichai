# Katichai Development Tasks

## Milestone 1: Core CLI Setup & Foundation ✅

### 1.1 Project Initialization
- [x] Initialize Go module (`go mod init`)
- [x] Set up project directory structure
  - [x] Create `cmd/katich/` directory
  - [x] Create `internal/` subdirectories (context, git, analysis, llm, review)
  - [x] Create `.katich/` directory for generated artifacts
- [x] Add `.gitignore` for Go projects
- [x] Create `README.md` with project overview

### 1.2 CLI Framework (Cobra)
- [x] Install Cobra CLI library (`github.com/spf13/cobra`)
- [x] Create root command in `cmd/katich/main.go`
- [x] Implement `katich version` command
- [x] Implement `katich doctor` command (health check)
- [x] Add global flags (--verbose, --config, etc.)

### 1.3 Context Commands Skeleton
- [x] Create `context` command group
- [x] Implement `katich context build` skeleton
- [x] Implement `katich context show` skeleton
- [x] Implement `katich context clear` skeleton
- [x] Add command flags and validation

### 1.4 Review Commands Skeleton
- [x] Create `review` command group
- [x] Implement `katich review latest` skeleton
- [x] Implement `katich review diff <range>` skeleton
- [x] Implement `katich review file <path>` skeleton
- [x] Implement `katich review --ci` flag
- [x] Add command flags and validation

### 1.5 Configuration Management
- [x] Create config file structure (`.katich/config.yaml`)
- [x] Implement config loader (`internal/config/`)
- [x] Support for LLM API keys (OpenAI, etc.)
- [x] Support for embedding model selection
- [x] Support for custom rules/patterns

---

## Milestone 2: Git Integration & Diff Extraction ✅

### 2.1 Git Repository Detection
- [x] Create `internal/git/repo.go`
- [x] Implement Git repository detection
- [x] Validate Git installation
- [x] Get repository root path

### 2.2 Diff Extraction
- [x] Create `internal/git/diff.go`
- [x] Implement diff extraction for latest commit
- [x] Implement diff extraction for commit range
- [x] Implement diff extraction for specific files
- [x] Parse diff output into structured format

### 2.3 Commit Analysis
- [x] Create `internal/git/commit.go`
- [x] Extract commit metadata (author, message, timestamp)
- [x] Get list of changed files
- [x] Get file content before/after changes
- [x] Handle binary files gracefully

---

## Milestone 3: Framework & Language Detection ✅

### 3.1 Language Detection
- [x] Create `internal/context/detector.go`
- [x] Implement file extension-based language detection
- [x] Support: Go, Java, Python, JavaScript, TypeScript, Rust, Kotlin, Swift, Ruby, PHP, C, C++, C#
- [x] Create language registry/enum

### 3.2 Framework Detection - Java/Spring Boot
- [x] Detect Maven (`pom.xml`) and Gradle (`build.gradle`)
- [x] Parse dependencies for Spring Boot
- [x] Detect annotations: `@SpringBootApplication`, `@RestController`, `@Service`
- [x] Identify Spring patterns (Controller → Service → Repository)

### 3.3 Framework Detection - Node.js/Express
- [x] Parse `package.json` for Express dependency
- [x] Detect Express patterns (`app.use()`, `express()`)
- [x] Identify middleware patterns

### 3.4 Framework Detection - Next.js/React
- [x] Detect `app/` or `pages/` directory structure
- [x] Parse `package.json` for Next.js/React
- [x] Detect `.tsx`/`.jsx` components
- [x] Identify React patterns (hooks, components)

### 3.5 Framework Detection - Python/FastAPI
- [x] Parse `requirements.txt` or `pyproject.toml`
- [x] Detect FastAPI imports and patterns
- [x] Detect Pydantic models
- [x] Identify route decorators

### 3.6 Framework Detection - Go/Gin
- [x] Parse `go.mod` for Gin dependency
- [x] Detect `gin.Default()` and router patterns
- [x] Identify handler patterns (`.GET`, `.POST`, `.PUT`)
- [x] Detect middleware usage

### 3.7 Framework Detection - Popular UI Frameworks
- [x] React, Vue.js, Angular, Svelte detection
- [x] SolidJS, Preact, Alpine.js detection
- [x] Meta frameworks: Next.js, Nuxt, Remix, SvelteKit, Gatsby, Astro
- [x] UI libraries: Material-UI, Ant Design, Chakra UI, Tailwind CSS, shadcn/ui, Mantine, Vuetify
- [x] Mobile frameworks: React Native
- [x] Build tools: Vite, Webpack

### 3.8 Framework Metadata Storage
- [x] Create framework metadata structure
- [x] Store detected frameworks in `context.json`
- [x] Include framework type (Backend, Frontend, FullStack, UI, Build)
- [x] Include language information

---

## Milestone 4: Static Analysis & AST Parsing ✅

### 4.1 AST Parser - Go
- [x] Create `internal/analysis/parser_go.go`
- [x] Use `go/parser` and `go/ast` packages
- [x] Extract functions, methods, structs
- [x] Extract imports and dependencies
- [x] Calculate cyclomatic complexity

### 4.2 Code Metrics Extraction
- [x] Create `internal/analysis/metrics.go`
- [x] Calculate lines of code (LOC)
- [x] Calculate function/method length
- [x] Detect nested complexity
- [x] Track function and class counts

### 4.3 Analysis Engine
- [x] Create `internal/analysis/analyzer.go`
- [x] Repository-wide analysis orchestration
- [x] Aggregate metrics across files
- [x] Top complexity and length tracking
- [x] Issue detection and categorization

### 4.4 Code Quality Detectors
- [x] Create `internal/analysis/detectors.go`
- [x] AI-generated code pattern detection
- [x] Generic naming detection
- [x] Excessive complexity warnings
- [x] Function length violations

### 4.5 Integration
- [x] Integrated analysis into `context build` command
- [x] Display code metrics in output
- [x] Show top complex functions
- [x] Report issues by severity
- [x] Save analysis results to context.json

---

## Milestone 5: Embeddings & Similarity Search

### 5.1 Embedding Model Integration
- [ ] Create `internal/context/embeddings.go`
- [ ] Research and select embedding model (Jina AI CodeV2, BGE Code, Nomic Embed, Snowflake Arctic)
- [ ] Implement model loading (local or API-based)
- [ ] Create embedding generation for code snippets
- [ ] Handle batching for large codebases

### 5.2 Code Chunking Strategy
- [ ] Create `internal/context/chunker.go`
- [ ] Implement function-level chunking
- [ ] Implement class-level chunking
- [ ] Handle large functions (split intelligently)
- [ ] Preserve context in chunks

### 5.3 FAISS Index Setup
- [ ] Install FAISS Go bindings or use CGo
- [ ] Create `internal/context/index.go`
- [ ] Initialize FAISS index
- [ ] Add embeddings to index
- [ ] Save index to `.katich/embeddings.index`
- [ ] Load index from disk

### 5.4 Similarity Search
- [ ] Create `internal/analysis/similarity.go`
- [ ] Implement k-NN search for similar code
- [ ] Set similarity threshold
- [ ] Return ranked results with scores
- [ ] Handle edge cases (empty index, no matches)

### 5.5 Context Building Pipeline
- [ ] Create `internal/context/builder.go`
- [ ] Orchestrate: scan → parse → embed → index
- [ ] Generate `context.json` with metadata
- [ ] Handle incremental updates (detect changed files)
- [ ] Add progress indicators for large repos

---

## Milestone 6: AI-Generated Code Detection

### 6.1 Heuristic Detection
- [ ] Create `internal/analysis/ai_detector.go`
- [ ] Detect overly verbose functions (LOC threshold)
- [ ] Detect repeated code blocks within function
- [ ] Detect generic naming patterns (Manager, Helper, Processor, UtilService)
- [ ] Detect unnecessary abstraction layers
- [ ] Detect framework pattern misuse

### 6.2 Code Drift Detection
- [ ] Compare new code style with repo norms
- [ ] Detect deviation in naming conventions
- [ ] Detect deviation in error handling patterns
- [ ] Detect deviation in import organization

### 6.3 Small LLM Classifier
- [ ] Create `internal/llm/classifier.go`
- [ ] Integrate small open-source LLM (e.g., CodeLlama, Phi-3)
- [ ] Create classification prompt template
- [ ] Classify code as:
  - [ ] `AI_GENERATED_BOILERPLATE`
  - [ ] `DUPLICATE_LOGIC`
  - [ ] `VALID_NEW_LOGIC`
  - [ ] `ARCHITECTURE_VIOLATION`
- [ ] Return classification with confidence score

---

## Milestone 7: Duplication Detection

### 7.1 Exact Duplication
- [ ] Create `internal/analysis/duplication.go`
- [ ] Implement hash-based exact match detection
- [ ] Detect copy-pasted code blocks
- [ ] Report file locations of duplicates

### 7.2 Semantic Duplication
- [ ] Use embedding similarity for semantic duplicates
- [ ] Set threshold for "similar enough" code
- [ ] Detect refactoring opportunities
- [ ] Suggest existing functions to reuse

### 7.3 Cross-Language Duplication
- [ ] Detect similar logic across languages (e.g., Go and TypeScript)
- [ ] Use embeddings for language-agnostic comparison

---

## Milestone 8: LLM Integration

### 8.1 LLM Client Setup
- [ ] Create `internal/llm/client.go`
- [ ] Support OpenAI API
- [ ] Support Anthropic Claude API
- [ ] Support local LLMs (Ollama, LM Studio)
- [ ] Implement retry logic and error handling

### 8.2 Prompt Engineering
- [ ] Create `internal/llm/prompts.go`
- [ ] Design system prompt for code review
- [ ] Create prompt template with context injection
- [ ] Include: diff, framework context, repo summary, similar code matches
- [ ] Optimize for concise, high-signal output

### 8.3 Review Synthesis
- [ ] Create `internal/review/synthesizer.go`
- [ ] Combine static analysis + AI detection + LLM reasoning
- [ ] Generate structured review output
- [ ] Include severity levels (info, warning, error)
- [ ] Include actionable suggestions

---

## Milestone 9: Review Engine

### 9.1 Review Orchestration
- [ ] Create `internal/review/engine.go`
- [ ] Orchestrate full review pipeline:
  1. [ ] Extract diff
  2. [ ] Load context
  3. [ ] Analyze modified functions
  4. [ ] Generate embeddings for new code
  5. [ ] Search for similar code
  6. [ ] Run static analysis
  7. [ ] Run AI detection
  8. [ ] Run LLM classifier
  9. [ ] Synthesize final review
- [ ] Handle errors gracefully at each step

### 9.2 Review Caching
- [ ] Cache LLM responses for identical diffs
- [ ] Cache embedding generation
- [ ] Implement cache invalidation strategy

### 9.3 CI Mode
- [ ] Implement `--ci` flag behavior
- [ ] Exit with non-zero code on critical issues
- [ ] Format output for CI systems (GitHub Actions, GitLab CI)
- [ ] Generate JSON output for parsing

---

## Milestone 10: Output Formatting

### 10.1 Terminal Output
- [ ] Create `internal/review/formatter.go`
- [ ] Implement colorized terminal output
- [ ] Use icons/emojis for severity levels
- [ ] Format code snippets with syntax highlighting
- [ ] Add file/line number references

### 10.2 Markdown Output
- [ ] Generate markdown report
- [ ] Include table of contents
- [ ] Format code blocks properly
- [ ] Add links to files (for GitHub)

### 10.3 JSON Output
- [ ] Generate structured JSON output
- [ ] Include all review findings
- [ ] Include metadata (timestamp, commit hash, etc.)
- [ ] Support `--output-format=json` flag

### 10.4 HTML Output (Optional)
- [ ] Generate standalone HTML report
- [ ] Include interactive elements
- [ ] Support `--output-format=html` flag

---

## Milestone 11: Testing & Quality

### 11.1 Unit Tests
- [ ] Write tests for context builder
- [ ] Write tests for framework detection
- [ ] Write tests for diff extraction
- [ ] Write tests for static analysis
- [ ] Write tests for duplication detection
- [ ] Write tests for AI detection heuristics
- [ ] Achieve >80% code coverage

### 11.2 Integration Tests
- [ ] Create test repositories for each framework
- [ ] Test end-to-end context building
- [ ] Test end-to-end review generation
- [ ] Test CI mode integration

### 11.3 Performance Testing
- [ ] Benchmark context building on large repos
- [ ] Benchmark embedding generation
- [ ] Benchmark FAISS search
- [ ] Optimize bottlenecks

### 11.4 Error Handling
- [ ] Add comprehensive error messages
- [ ] Handle missing Git installation
- [ ] Handle missing LLM API keys
- [ ] Handle corrupted context files
- [ ] Handle network failures gracefully

---

## Milestone 12: Documentation & Polish

### 12.1 User Documentation
- [ ] Write comprehensive README.md
- [ ] Create installation guide
- [ ] Create usage examples
- [ ] Document all CLI commands and flags
- [ ] Create troubleshooting guide

### 12.2 Developer Documentation
- [ ] Document architecture and design decisions
- [ ] Create contribution guide
- [ ] Document how to add new framework support
- [ ] Document how to add new language support

### 12.3 Examples & Demos
- [ ] Create example repositories
- [ ] Create demo videos/GIFs
- [ ] Create blog post or tutorial

### 12.4 Release Preparation
- [ ] Set up GitHub releases
- [ ] Create changelog
- [ ] Set up versioning strategy (semantic versioning)
- [ ] Create installation scripts (Homebrew, apt, etc.)

---

## Future Enhancements (Post-MVP)

### Advanced Features
- [ ] Support for more languages (Rust, C++, Ruby, PHP)
- [ ] Support for more frameworks (Django, Rails, Laravel, etc.)
- [ ] Custom rule engine (user-defined patterns)
- [ ] Integration with GitHub/GitLab (PR comments)
- [ ] Web UI for reviewing reports
- [ ] Team analytics (track code quality over time)
- [ ] Auto-fix suggestions (with user approval)

### Performance Optimizations
- [ ] Parallel processing for large repos
- [ ] Incremental context updates
- [ ] Distributed embedding generation
- [ ] GPU acceleration for embeddings

### Enterprise Features
- [ ] Self-hosted LLM support
- [ ] SSO integration
- [ ] Audit logs
- [ ] Custom compliance rules
- [ ] Multi-repo analysis

---

## Notes
- Tasks marked with `[ ]` are pending
- Tasks marked with `[/]` are in progress
- Tasks marked with `[x]` are completed
- Dependencies between tasks should be respected (e.g., can't do embeddings without AST parsing)
