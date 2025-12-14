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
- [x] Create `internal/context/embeddings.go` (done as `internal/embeddings/provider.go`)
- [x] Research and select embedding model (Jina AI CodeV2, BGE Code, Nomic Embed, Snowflake Arctic)
- [x] Implement model loading (local or API-based)
- [x] Create embedding generation for code snippets
- [x] Handle batching for large codebases

### 5.2 Code Chunking Strategy
- [x] Create `internal/context/chunker.go` (integrated in generator)
- [x] Implement function-level chunking
- [ ] Implement class-level chunking
- [x] Handle large functions (split intelligently)
- [x] Preserve context in chunks

### 5.3 FAISS Index Setup
- [x] Install FAISS Go bindings or use CGo (Simulated with simple JSON for now)
- [x] Create `internal/context/index.go` (as `internal/embeddings/generator.go`)
- [x] Initialize FAISS index
- [x] Add embeddings to index
- [x] Save index to `.katich/embeddings.index`
- [x] Load index from disk

### 5.4 Similarity Search
- [x] Create `internal/analysis/similarity.go` (as `internal/embeddings/similarity.go`)
- [x] Implement k-NN search for similar code
- [x] Set similarity threshold
- [x] Return ranked results with scores
- [x] Handle edge cases (empty index, no matches)

### 5.5 Context Building Pipeline
- [x] Create `internal/context/builder.go` (part of commands)
- [x] Orchestrate: scan → parse → embed → index
- [x] Generate `context.json` with metadata
- [x] Implement incremental updates
  - Only process changed files
  - Update/Delete embeddings for modified filespos
- [x] Add progress indicators for large repos

---

## Milestone 6: AI-Generated Code Detection

### 6.1 Heuristic Detection
### 6.1 Heuristic Detection
- [x] Create `internal/analysis/ai_detector.go`
- [x] Detect overly verbose functions (LOC threshold)
- [x] Detect repeated code blocks within function
- [x] Detect generic naming patterns (Manager, Helper, Processor, UtilService)
- [x] Detect unnecessary abstraction layers
- [x] Detect framework pattern misuse

### 6.2 Code Drift Detection
- [x] Compare new code style with repo norms (Basic check implemented)
- [x] Detect deviation in naming conventions
- [x] Detect deviation in error handling patterns
- [ ] Detect deviation in import organization

### 6.3 Small LLM Classifier
- [x] Create `internal/llm/classifier.go`
- [x] Integrate small open-source LLM (or use main model with classification prompt)
- [x] Create classification prompt template
- [x] Classify code as: BOILERPLATE, LOGIC, REFACTOR, etc.
- [x] Return classification with confidence score

---

## Milestone 7: Duplication Detection

### 7.1 Exact Duplication
- [x] Create `internal/analysis/duplication.go`
- [x] Implement hash-based exact match detection
- [x] Detect copy-pasted code blocks
- [x] Report file locations of duplicates

### 7.2 Semantic Duplication
- [x] Use embedding similarity for semantic duplicates (Backend ready)
- [x] Set threshold for "similar enough" code (Distinct thresholds for dup vs reuse)
- [x] Detect refactoring opportunities
- [x] Suggest existing functions to reuse

### 7.3 Cross-Language Duplication
- [x] Detect similar logic across languages (Implicitly supported via embeddings)
- [x] Use embeddings for language-agnostic comparison

---

## Milestone 8: LLM Integration

### 8.1 LLM Client Setup
- [x] Create `internal/llm/client.go`
- [x] Support OpenAI API
- [x] Support Anthropic Claude API
- [x] Support local LLMs (Ollama, LM Studio)
- [x] Implement retry logic and error handling (Basic timeout implemented)

### 8.2 Prompt Engineering
- [x] Create `internal/llm/prompts.go`
- [x] Design system prompt for code review
- [x] Create prompt template with context injection
- [x] Include: diff, framework context, repo summary, similar code matches
- [x] Optimize for concise, high-signal output

### 8.3 Review Synthesis
- [x] Create `internal/review/synthesizer.go`
- [x] Combine static analysis + AI detection + LLM reasoning
- [x] Generate structured review output
- [x] Include severity levels (info, warning, error)
- [x] Include actionable suggestions
- [x] Support multi-format output (Text, JSON, Markdown)

---

## Milestone 9: Review Engine

- [ ] Handle errors gracefully at each step

... (Remaining milestones omitted for brevity)
