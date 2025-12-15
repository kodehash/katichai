# Generic Regex Parser Implementation Plan

## Goal Description
Fix the issue where non-Go languages (like Java) have 0 function metrics and 0 embeddings. Implement a regex-based parser to extract function/method definitions from Java, Python, JavaScript, and TypeScript files.

## Proposed Changes

### 1. Create Generic Parsers
#### [NEW] [internal/analysis/parser_regex.go](file:///Users/ko/webknot/code/katich-ai/internal/analysis/parser_regex.go)
- Define `RegexParser` struct.
- Implement `ParseFile(path string) (*FileAnalysis, error)`.
- Support multiple languages with regex patterns:
  - **Java/C#**: `(public|private|protected|static|\s) +[\w\<\>\[\]]+\s+(\w+) *\([^\)]*\) *(\{?|[^;])`
  - **Python**: `def\s+(\w+)\s*\(`
  - **JS/TS**: `function\s+(\w+)|const\s+(\w+)\s*=\s*\(|(\w+)\s*\([^)]*\)\s*\{`
- Extract function body by counting braces (for C-style languages) or indentation (for Python - maybe skipping body extraction for Python initially or doing simple block detection).
- Calculate basic SHA256 hash.

### 2. Integration
#### [MODIFY] [internal/analysis/analyzer.go](file:///Users/ko/webknot/code/katich-ai/internal/analysis/analyzer.go)
- In `analyzeFile`, add cases for `LanguageJava`, `LanguageJavaScript`, `LanguageTypeScript`, `LanguagePython`.
- Instantiate `RegexParser` with the appropriate language configuration.

## Verification Plan
### Automated Tests
- Create `parser_regex_test.go` with sample Java and JS code.
- Verify function usage counts and name extraction.

### Manual Verification
- (User side) Run `katich context build` on the Java repo again and check metrics.
