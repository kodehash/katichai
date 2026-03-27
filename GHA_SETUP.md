# Katich AI GitHub Actions Setup

This guide shows how to integrate Katich AI into GitHub Actions with:

- PR-time review on changed code
- Context build before review
- Optional remote context publish on trusted events

## 1) Prerequisites

Before enabling workflows, ensure:

- Your repository has Katich config at `.katich/config.yaml` (or uses defaults from `.katich/config.example.yaml`)
- You have an LLM API key configured in GitHub Secrets
- Your branch/repo permissions allow artifact upload

Recommended secrets:

- `OPENAI_API_KEY` (or provider-specific key via `KATICH_LLM_API_KEY`)

## 2) PR Review Workflow (Context Build + Diff Review)

Create `.github/workflows/katich-pr-review.yml`:

```yaml
name: Katich PR Review

on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  contents: read

jobs:
  katich-review:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build Katich
        run: go build -o katich ./cmd/katich

      - name: Initialize Katich
        run: ./katich init

      - name: Build Context (PR)
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          KATICH_LLM_API_KEY: ${{ secrets.KATICH_LLM_API_KEY }}
        run: ./katich context build --incremental

      - name: Run Diff Review
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          KATICH_LLM_API_KEY: ${{ secrets.KATICH_LLM_API_KEY }}
        run: |
          BASE_SHA="${{ github.event.pull_request.base.sha }}"
          HEAD_SHA="${{ github.event.pull_request.head.sha }}"
          ./katich review diff "${BASE_SHA}..${HEAD_SHA}" --gfm

      - name: Add Report to Job Summary
        shell: bash
        run: |
          latest_report="$(ls -1t .katich/reports/*.md 2>/dev/null | head -n 1 || true)"
          if [ -n "$latest_report" ] && [ -f "$latest_report" ]; then
            {
              echo "## Katich Review Report"
              echo
              cat "$latest_report"
            } >> "$GITHUB_STEP_SUMMARY"
          else
            echo "## Katich Review Report" >> "$GITHUB_STEP_SUMMARY"
            echo "No markdown report was generated. Check logs/artifacts." >> "$GITHUB_STEP_SUMMARY"
          fi

      - name: Upload Katich Reports
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: katich-reports-${{ github.run_id }}
          path: .katich/reports/
          if-no-files-found: warn
```

## 3) Trusted Context Publish Workflow (Optional)

Publish remote context only from trusted events (for example, pushes to `main`), not from untrusted PRs/forks.

Create `.github/workflows/katich-context-publish.yml`:

```yaml
name: Katich Context Publish

on:
  push:
    branches:
      - main
  workflow_dispatch:

permissions:
  contents: write

jobs:
  publish-context:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build Katich
        run: go build -o katich ./cmd/katich

      - name: Initialize Katich
        run: ./katich init

      - name: Build and Publish Context
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          KATICH_LLM_API_KEY: ${{ secrets.KATICH_LLM_API_KEY }}
          KATICH_API_SERVER_URL: ${{ secrets.KATICH_API_SERVER_URL }}
          KATICH_API_TOKEN: ${{ secrets.KATICH_API_TOKEN }}
        run: ./katich context build --incremental --publish
```

## 4) Safety and CI Behavior Notes

- Do not publish remote context from `pull_request` workflows that may run untrusted code.
- `katich review full` is interactive; avoid it in CI.
- Use `katich review diff base..head` for deterministic PR analysis.
- Current `--ci` behavior does not enforce findings-based failures by itself; workflow fails reliably on runtime/config/command errors.

## 5) Recommended Rollout

1. Add `katich-pr-review.yml` first and verify report generation.
2. Confirm API keys and context build stability in CI.
3. Add `katich-context-publish.yml` for trusted branch context refresh.
4. Optionally add policy logic later (for example, fail on specific findings parsed from JSON/Markdown output).

