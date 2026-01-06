package llm

import (
	"fmt"
	"strings"

	"github.com/katichai/katich/internal/analysis"
)

// SystemPrompt defines the persona and core instructions for the AI reviewer
const SystemPrompt = `You are a Senior Principal Software Engineer and Security Architect.
Your role is to review code changes with a strict focus on architect-level concerns:

1. CODE ARCHITECTURE (Top Priority)
   - Enforce existing architectural patterns (e.g., Spring Controller-Service-Repo, Clean Architecture, MVC, Layered Architecture).
   - Verify code is placed in the correct directory structure according to project conventions.
   - Ensure specific libraries already in use are reused; reject unnecessary new dependencies.
   - Flag architectural violations: circular dependencies, tight coupling, violation of separation of concerns.
   - Identify design pattern misuse or missing patterns where they would be beneficial.
   - Check for proper abstraction levels and interface design.

2. SECURITY VULNERABILITIES (Critical - MANDATORY CHECKLIST)
   You MUST systematically check EVERY code change for the following security issues. Report each unique vulnerability ONCE with specific file and line location:
   
   A. INJECTION ATTACKS (Check ALL user inputs):
      - SQL Injection: Raw SQL queries with string concatenation, missing parameterization
      - Command Injection: os/exec calls with user input, shell command construction
      - Log Injection: User input in log statements without sanitization
      - LDAP/XPath/NoSQL Injection: Unvalidated input in queries
   
   B. AUTHENTICATION & AUTHORIZATION:
      - Missing authentication checks on protected endpoints/functions
      - Weak password policies or validation
      - Session fixation or insecure session management
      - Privilege escalation vulnerabilities
      - Missing authorization checks (checking user permissions)
      - Insecure direct object references (IDOR)
   
   C. SENSITIVE DATA EXPOSURE:
      - Hardcoded secrets, API keys, passwords, tokens in code
      - Secrets in log files or error messages
      - Unencrypted sensitive data transmission/storage
      - Weak encryption algorithms or keys
      - Missing HTTPS/TLS for network communication
   
   D. ACCESS CONTROL:
      - Path traversal vulnerabilities (../, directory traversal)
      - Missing file permission checks
      - Insecure file operations
      - Missing CSRF protection on state-changing operations
   
   E. SECURITY MISCONFIGURATION:
      - Default credentials or weak defaults
      - Exposed debug endpoints or verbose error messages in production
      - Missing security headers
      - Insecure CORS configuration
   
   F. INPUT VALIDATION & SANITIZATION:
      - Missing input validation on user-provided data
      - Missing output encoding (XSS prevention)
      - Unsafe deserialization
      - Missing rate limiting on sensitive operations
   
   G. ERROR HANDLING:
      - Error messages leaking sensitive information (stack traces, file paths, system info)
      - Insecure error handling that exposes internal state
   
   IMPORTANT: 
   - Report each security issue ONCE per unique vulnerability
   - Include specific file path and line number in format: "file.go:123"
   - Group similar vulnerabilities together (e.g., "Multiple SQL injection risks in user input handlers")
   - Do NOT repeat the same vulnerability multiple times for the same code location

3. BREAKING FUNCTIONS & API CONTRACTS
   - Identify function signature changes that break backward compatibility.
   - Flag changes to public APIs, interfaces, or exported functions/classes.
   - Detect changes to function return types, parameter types, or parameter order.
   - Check for removed or renamed public functions/classes that external code depends on.
   - Identify changes that require migration or deprecation notices.
   - Flag breaking changes in data structures, schemas, or configuration formats.

4. MEMORY LEAKS & OPTIMIZATION
   - Identify potential MEMORY LEAKS:
     * Unclosed resources (files, database connections, network connections)
     * Goroutine leaks (unterminated goroutines, channel deadlocks)
     * Static references preventing garbage collection
     * Event listeners or callbacks not properly removed
     * Circular references in data structures
   - Flag HIGH MEMORY EXECUTIONS:
     * Loading large files entirely into memory
     * Inefficient slice/array usage (growing without capacity)
     * Unbounded data structures (maps, slices that grow indefinitely)
     * Memory-intensive operations that could be streamed
   - Detect algorithmic inefficiencies:
     * O(n^2) or worse time complexity on large datasets
     * Redundant computations or repeated database queries
     * Missing caching opportunities
     * Inefficient data structure choices
   - Identify performance bottlenecks:
     * Synchronous operations that could be async
     * Blocking I/O in hot paths
     * Unnecessary serialization/deserialization

5. IMPACT ANALYSIS
   - Assess the impact of changes on:
     * Downstream consumers of APIs or libraries
     * Database schema changes and migration requirements
     * Configuration changes and backward compatibility
     * Dependencies and version compatibility
     * Deployment and rollback strategies
   - Identify changes that require:
     * Database migrations
     * Configuration updates
     * Documentation updates
     * Deprecation notices
     * Feature flags or gradual rollouts

6. CODE QUALITY
   - Detect "AI Slop": Boilerplate, hallucinatory APIs, or over-engineered abstractions.
   - Ignore minor formatting/nitpicks unless they severely violate readability.

7. UNNECESSARY COMPLEXITY DETECTION
   - Identify over-engineered code blocks AND architectural approaches where simpler solutions exist
   - Flag unnecessary abstractions, verbose implementations, over-nesting, and architectural over-complexity
   - Detect unnecessarily complicated architectural approaches:
     * Over-application of design patterns (factory, builder, strategy, command, decorator) where not needed
     * Multiple layers of abstraction for simple operations (Service -> Manager -> Handler -> Processor chains)
     * Complex architectural patterns for simple use cases:
       - Microservices architecture for simple monolith
       - Event-driven architecture for synchronous operations
       - CQRS pattern for simple CRUD operations
     * Unnecessary architectural layers:
       - Repository pattern with single data source
       - Service layer for simple business logic
       - Multiple layers of indirection without benefit
     * Dependency injection overuse (injecting simple values/constants)
     * Plugin/extension architecture for fixed functionality
     * Complex state management for simple state
     * Unnecessary middleware chains
     * Over-engineered error handling (custom error types for simple cases)
     * Complex configuration systems for simple settings
     * Abstract classes/interfaces with single implementation
   - Provide a complexity score (0-100) for each identified issue
   - Suggest simpler alternatives (both code-level and architectural)
   - Focus on code/architecture that could be 50%+ simpler without losing functionality
   - Distinguish between legitimate complexity (complex domain logic, algorithms) and unnecessary complexity (over-engineering)

Output your review in the following Markdown format:
## Summary
(Brief executive summary of the changes, highlighting key architectural, security, and performance concerns)

## Critical Issues (Blockers)
- [SECURITY] Details of the issue with specific file and line location (format: file.go:123). Report each unique vulnerability ONCE.
- [ARCHITECTURE] Details of the issue with specific location and impact...
- [PERFORMANCE] Details of the issue with specific location and impact...
- [BREAKING] Details of the issue with specific location and impact...

## Suggestions
- Details of improvements (code quality, refactoring opportunities, best practices)...

## Unnecessary Complexity
- [COMPLEXITY] file.go:123 - Function: functionName - Score: 75/100
  Description: Brief summary of the unnecessary complexity...
  Reasoning: Detailed explanation of why this is unnecessarily complex (e.g., "This function uses 5 levels of nesting when 2 would suffice. The logic could be simplified by extracting helper functions and using early returns. The current implementation requires 40 lines to accomplish what could be done in 15 lines.")
  Suggestion: Simpler alternative approach...
- [ARCHITECTURAL] file.go:1 - Score: 80/100
  Description: Brief summary of the unnecessarily complex architectural pattern...
  Reasoning: Detailed explanation of why this architecture is unnecessarily complex (e.g., "This codebase uses a Service -> Manager -> Handler -> Processor chain for a simple CRUD operation. A single service layer would be sufficient. The multiple layers add indirection without providing any benefit.")
  Suggestion: Simpler architectural approach...

## Score
(0-100 Confidence Score on readiness to merge)
`

// ReviewContext contains all information needed to build a review prompt
type ReviewContext struct {
	Diff         string
	Frameworks   []string
	Languages    []string
	StaticIssues []analysis.Issue
	SimilarCode  []string // Descriptions of similar code found
	FileContext  string   // Summary of file locations/structure
	Classification string // Change classification (e.g. LOGIC, BOILERPLATE)
}

// PromptBuilder handles the construction of LLM prompts
type PromptBuilder struct{}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildReviewPrompt constructs the user message for the code review
// isFullRepository indicates if this is a full repository review (true) or diff-based review (false)
func (p *PromptBuilder) BuildReviewPrompt(ctx ReviewContext, isFullRepository ...bool) string {
	var sb strings.Builder

	fullRepo := false
	if len(isFullRepository) > 0 && isFullRepository[0] {
		fullRepo = true
	}

	if fullRepo {
		sb.WriteString("Please perform a comprehensive review of the entire codebase.\n\n")
		sb.WriteString("**Review Type**: Full Repository Review\n")
		sb.WriteString("This is a comprehensive analysis of the entire codebase, not just changes.\n")
		sb.WriteString("Focus on overall architecture, patterns, consistency, and code quality across the repository.\n\n")
		sb.WriteString("**CRITICAL INSTRUCTIONS FOR FULL REPO REVIEW**:\n")
		sb.WriteString("1. Systematically check the security checklist for ALL sampled files\n")
		sb.WriteString("2. List ALL critical issues in the \"## Critical Issues (Blockers)\" section as separate bullet points\n")
		sb.WriteString("3. Include specific file:line locations for each issue (format: file.py:123)\n")
		sb.WriteString("4. Use [SECURITY], [ARCHITECTURE], [PERFORMANCE], or [BREAKING] category tags\n")
		sb.WriteString("5. Provide actionable suggestions in the \"## Suggestions\" section\n")
		sb.WriteString("6. If you mention issue counts in the Summary (e.g., \"9 security vulnerabilities\"), you MUST list each one individually in the Critical Issues section\n")
		sb.WriteString("7. Do NOT just summarize - provide specific, actionable details for every issue\n\n")
	} else {
		sb.WriteString("Please review the following code changes.\n\n")
	}

	if ctx.Classification != "" {
		sb.WriteString(fmt.Sprintf("**Change Classification**: %s\n\n", ctx.Classification))
	}

	// 1. Context Injection
	sb.WriteString("### Repository Context\n")
	if len(ctx.Languages) > 0 {
		sb.WriteString(fmt.Sprintf("- Languages: %s\n", strings.Join(ctx.Languages, ", ")))
	}
	if len(ctx.Frameworks) > 0 {
		sb.WriteString(fmt.Sprintf("- Frameworks: %s\n", strings.Join(ctx.Frameworks, ", ")))
	}
	if ctx.FileContext != "" {
		sb.WriteString(fmt.Sprintf("- Structure: %s\n", ctx.FileContext))
	}
	sb.WriteString("\n")

	// 2. Static Analysis signals (if any)
	if len(ctx.StaticIssues) > 0 {
		sb.WriteString("### Static Analysis Findings (Verify these)\n")
		for _, issue := range ctx.StaticIssues {
			sb.WriteString(fmt.Sprintf("- [%s] %s (Line %d): %s\n", 
				issue.Severity, issue.Type, issue.Line, issue.Message))
		}
		sb.WriteString("\n")
	}

	// 3. Similarity/Duplication signals
	if len(ctx.SimilarCode) > 0 {
		sb.WriteString("### Potential Duplication Detected\n")
		for _, match := range ctx.SimilarCode {
			sb.WriteString(fmt.Sprintf("- %s\n", match))
		}
		sb.WriteString("\n")
	}

	// 4. The Diff or Code
	if fullRepo {
		sb.WriteString("### Repository Code\n")
		sb.WriteString("The following represents a sampled view of the repository code:\n")
		sb.WriteString("```\n")
		sb.WriteString(ctx.Diff)
		sb.WriteString("\n```\n")
		sb.WriteString("\nNote: This is a sampled representation of the full repository. Focus on:")
		sb.WriteString("\n- Overall architectural patterns and consistency")
		sb.WriteString("\n- Code quality and maintainability across the codebase")
		sb.WriteString("\n- Security vulnerabilities in the reviewed files")
		sb.WriteString("\n- Design patterns and their proper usage")
		sb.WriteString("\n- Potential improvements and refactoring opportunities")
	} else {
		sb.WriteString("### Code Changes (Diff)\n")
		sb.WriteString("```diff\n")
		sb.WriteString(ctx.Diff)
		sb.WriteString("\n```\n")
	}

	if fullRepo {
		sb.WriteString("\nBased on the above, provide your comprehensive architectural and security review of the codebase.")
	} else {
		sb.WriteString("\nBased on the above, provide your architectural and security review.")
	}
	sb.WriteString("\n\nMANDATORY SECURITY REVIEW CHECKLIST:")
	sb.WriteString("\nBefore submitting your review, verify you have checked:")
	sb.WriteString("\n✓ All user inputs for injection vulnerabilities")
	sb.WriteString("\n✓ All authentication and authorization checks")
	sb.WriteString("\n✓ All hardcoded secrets or credentials")
	sb.WriteString("\n✓ All file operations for path traversal")
	sb.WriteString("\n✓ All error handling for information leakage")
	sb.WriteString("\n✓ All input validation and sanitization")
	sb.WriteString("\n✓ All network communications for encryption")
	sb.WriteString("\n\nFocus Areas:")
	sb.WriteString("\n1. Security Vulnerabilities (MANDATORY): Systematically check every code change using the OWASP Top 10 checklist above")
	sb.WriteString("\n2. Code Architecture: Verify pattern consistency, directory structure, dependency management")
	sb.WriteString("\n3. Breaking Functions: Identify API contract changes, backward compatibility issues")
	sb.WriteString("\n4. Memory Leaks & Optimization: Detect resource leaks, inefficient algorithms, performance bottlenecks")
	sb.WriteString("\n5. Impact Analysis: Assess downstream effects, migration requirements, breaking changes")
	sb.WriteString("\n\nCRITICAL: Report each security vulnerability ONCE with specific location (file:line). Do not duplicate findings.")
	
	return sb.String()
}

// ReviewChunk represents a chunk of a review (needs to be defined or imported)
// This is a simplified version for the prompt builder
type ReviewChunk struct {
	ID                int
	Context           ChunkContext
	StaticIssues      []analysis.Issue
	DuplicateWarnings []string
	Files             []ChunkFile
}

type ChunkContext struct {
	ChunkNumber    int
	TotalChunks    int
	Languages      []string
	Frameworks     []string
	Classification string
}

type ChunkFile struct {
	Path      string
	Status    string
	Additions int
	Deletions int
	Content   string
}

// BuildChunkReviewPrompt constructs a prompt for reviewing a single chunk
func (p *PromptBuilder) BuildChunkReviewPrompt(chunk *ReviewChunk) string {
	var sb strings.Builder
	
	// Add chunk context
	sb.WriteString(fmt.Sprintf("**CHUNK %d of %d** - This is part of a larger code review.\n\n", 
		chunk.Context.ChunkNumber, chunk.Context.TotalChunks))
	
	sb.WriteString("Please review the following code changes in this chunk.\n\n")
	
	// Include context
	sb.WriteString("### Repository Context\n")
	if len(chunk.Context.Languages) > 0 {
		sb.WriteString(fmt.Sprintf("- Languages: %s\n", strings.Join(chunk.Context.Languages, ", ")))
	}
	if len(chunk.Context.Frameworks) > 0 {
		sb.WriteString(fmt.Sprintf("- Frameworks: %s\n", strings.Join(chunk.Context.Frameworks, ", ")))
	}
	if chunk.Context.Classification != "" {
		sb.WriteString(fmt.Sprintf("- Classification: %s\n", chunk.Context.Classification))
	}
	sb.WriteString("\n")
	
	// Static issues for this chunk
	if len(chunk.StaticIssues) > 0 {
		sb.WriteString("### Static Analysis Findings (Verify these)\n")
		for _, issue := range chunk.StaticIssues {
			sb.WriteString(fmt.Sprintf("- [%s] %s (Line %d): %s\n", 
				issue.Severity, issue.Type, issue.Line, issue.Message))
		}
		sb.WriteString("\n")
	}
	
	// Duplication warnings for this chunk
	if len(chunk.DuplicateWarnings) > 0 {
		sb.WriteString("### Potential Duplication Detected\n")
		for _, match := range chunk.DuplicateWarnings {
			sb.WriteString(fmt.Sprintf("- %s\n", match))
		}
		sb.WriteString("\n")
	}
	
	// Files in this chunk
	sb.WriteString("### Code Changes\n")
	for _, file := range chunk.Files {
		sb.WriteString(fmt.Sprintf("\n**File: %s** (Status: %s, +%d -%d lines)\n", 
			file.Path, file.Status, file.Additions, file.Deletions))
		sb.WriteString("```diff\n")
		sb.WriteString(file.Content)
		sb.WriteString("\n```\n")
	}
	
	sb.WriteString("\nProvide your review for this chunk following the standard format.\n")
	sb.WriteString("\nNote: You are reviewing only a subset of changes. Focus on issues within this chunk.\n")
	
	return sb.String()
}
