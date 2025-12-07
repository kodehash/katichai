KATICHAI — Combined PRD + Technical Design Document

Version: 1.0
Language: Go
Maintainer: Pramod / Katichai Engineering
Target Consumer: Antigravity (LLM-based code-generation agent)

--------------------------------------------------
📄 1. PRODUCT REQUIREMENTS DOCUMENT (PRD)
--------------------------------------------------
1.1 Product Name

Katichai

A context-aware, AI-assisted code review CLI tool that prevents unnecessary AI-generated code, detects duplicated logic, enforces architectural patterns, and ensures high-quality engineering standards.

1.2 Product Summary

Katichai is a CLI tool that:

Builds a semantic understanding of a codebase using static analysis, embeddings, and framework detection.

Performs code reviews on git diffs using hybrid AI (static analysis + open-source models + LLM).

Detects unnecessary AI-generated boilerplate.

Identifies duplicate logic and suggests reuse.

Enforces architecture and framework-specific conventions.

1.3 Problem Statement

Engineers now use AI tools (Cursor, Copilot, Antigravity) to generate code. This often results in:

Repeated logic

Overly verbose or bloated functions

Unnecessary abstraction layers

Architectural violations

Divergence from existing patterns

Wrong framework usage

Missing reuse of existing utilities

Katichai solves these issues at the diff level with full codebase context.

1.4 Goals
Core Goals

Build a contextual understanding of a codebase

Detect AI-generated unnecessary code

Identify redundant or duplicate logic

Detect framework choice and enforce conventions

Provide architecture-aware, project-aware code reviews

Work across languages: Java, Python, Go, JavaScript, TypeScript, React, Next.js, etc.

Secondary Goals

Minimize LLM usage (hybrid model)

Execute locally (offline-first)

Integrate with Git and CI

1.5 Non-Goals

Not auto-fixing code

Not replacing human code review

Not a linting tool

Not a cloud service (CLI-only)

1.6 User Personas
Software Engineers

Need strong AI guardrails and smart reviews.

Tech Leads / Architects

Need architecture protection and code consistency.

DevOps / CI Engineers

Need automated reviews in CI pipelines.

AI-heavy Developers

Need a tool to prevent AI hallucination in code.

1.7 User Workflow
1) Build codebase context
katich context build

2) Review a commit
katich review latest

3) Review any diff
katich review diff HEAD~3..HEAD

4) CI Mode
katich review --ci

--------------------------------------------------
📘 2. TECHNICAL DESIGN DOCUMENT (TDD)
--------------------------------------------------
2.1 High-Level Architecture
Katich CLI
 ├── Context Engine
 │     ├── Framework Detector
 │     ├── Static Analyzer
 │     ├── Embedding Generator
 │     └── Repo Signature Builder
 │
 ├── Review Engine
 │     ├── Diff Extractor
 │     ├── Duplicate/Reuse Detector
 │     ├── AI-Generated Code Detector
 │     ├── Similarity Search (FAISS/local index)
 │     ├── Small LLM Classifier
 │     └── Final LLM Review Synthesizer
 │
 ├── LLM Layer
 │     ├── Local open-source LLMs (classification)
 │     └── Cloud LLMs (final reasoning)
 │
 └── Output Formatter

2.2 Internal Components Overview
A. Context Generation Engine (katich context build)
Responsibilities:

Scan repository

Detect languages & frameworks

Parse ASTs

Extract functions, classes, imports

Generate per-function embeddings

Build FAISS similarity index

Construct context file: .katich/context.json

Output example:
{
  "languages": ["go", "typescript"],
  "frameworks": ["gin", "nextjs"],
  "modules": {
    "services/order_service.go": {
      "functions": ["CreateOrder", "CancelOrder"],
      "patterns": ["error wrapping", "repository delegation"]
    }
  },
  "patterns": ["controller -> service -> repo"],
  "embeddings": ["..."]
}

B. Review Engine (katich review)
Responsibilities:

Extract diff

Analyze modified functions

Embed new code

Compare with entire codebase embeddings

Detect duplication

Detect unnecessary abstractions

Detect AI-generated patterns

Perform static analysis (complexity, repetition, unused logic)

Use small LLM classification

Use big LLM for final review

2.3 Framework Detection

Katich identifies frameworks using:

Java / Spring Boot

@SpringBootApplication, @RestController, @Service

Maven/Gradle dependencies

Node.js / Express

express(), app.use()

package.json dependencies

Next.js / React

app/ or pages/ folder

.tsx components

Python / FastAPI

FastAPI()

Pydantic models

Go / Gin

gin.Default()

.GET, .POST, .PUT handlers

This metadata feeds into architecture reasoning.

2.4 Embedding Model

Use open-source embedding models:

Jina AI CodeV2

BGE Code

Nomic Embed

Snowflake Arctic Embed

Stored in:

.katich/embeddings.index


Similarity search using FAISS.

2.5 AI-Generated Code Detection

Heuristics detect:

Long, overly verbose functions

Repeated blocks

Generic names (Manager, Helper, Processor, UtilService)

Abstraction layers without purpose

Misuse of framework patterns

Code drift from repo norms

Small LLM classifier categorizes:

AI_GENERATED_BOILERPLATE

DUPLICATE_LOGIC

VALID_NEW_LOGIC

ARCHITECTURE_VIOLATION

2.6 Final Review Generation (LLM)

Input includes:

Diff

Framework context

Repo summary

Similar code matches

Static heuristics

AI generation classifier output

Output:

Duplication warnings

Framework violations

Architecture problems

Suggestions for code reuse

Recommendations for simplification

High-signal, concise review comments

2.7 CLI Command Specification (Go)
Context Commands
katich context build
katich context show
katich context clear

Review Commands
katich review latest
katich review diff <range>
katich review file <path>
katich review --ci

Utility Commands
katich doctor
katich version

2.8 Project Directory Structure
katichai/
  cmd/
    katich/main.go
  internal/
    context/
      builder.go
      detector.go
      embeddings.go
      store.go
    git/
      diff.go
      commit.go
    analysis/
      static.go
      duplication.go
      ai_detector.go
      similarity.go
    llm/
      openai.go
      classifier.go
    review/
      engine.go
      formatter.go
  .katich/
    context.json
    embeddings.index
  go.mod
  go.sum

2.9 Milestones / Roadmap
Milestone 1 — Core CLI

Cobra CLI setup

Context builder skeleton

Diff extraction

Milestone 2 — Embeddings / FAISS

Embedding generator

FAISS index

Similarity search

Milestone 3 — Framework Detection
Milestone 4 — Static Analysis + AI Detector
Milestone 5 — Full Review Engine
2.10 Non-Functional Requirements

Must run locally on macOS/Linux/Windows

Must work for large monorepos

Must minimize LLM calls

Deterministic local analysis

Extensible framework support