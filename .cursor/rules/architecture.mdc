---
description: 
globs: 
alwaysApply: true
---
# Architecture Principles

This document outlines the fundamental architectural principles and patterns for the `mcp-debug` codebase. Adherence to these guidelines is **mandatory** to maintain a clean, decoupled, and testable system.

---

## 1. Testing Standards

-   **Minimum Coverage:** All new code **MUST** achieve at least **80% unit test coverage**.
-   **No Flaky Tests:** Tests **MUST** be deterministic. Do not use `time.Sleep` or other mechanisms that can lead to race conditions. Use dependency injection for things like clocks or timers to make code testable.

---

## 2. Go Development & Style

-   **Formatting:** Before **every commit**, run:
    ```bash
    goimports -w .
    go fmt ./...
    ```
-   **Dependencies:** Before adding a new dependency with `go get`, use `web_search` to find its latest stable version.
-   **Error Handling:** Wrap errors with `fmt.Errorf("context: %w", err)` to preserve context.
-   **File Size:** Keep files focused and under **500 lines**. Refactor larger files.

---

## 3. Documentation

-   **Packages:** Every package **MUST** have a `doc.go` file explaining its purpose.
-   **Exported Members:** Every exported function, type, and variable **MUST** have a GoDoc comment.
-   **Project Docs:** New features or significant changes require updates to the relevant documents in the `docs/` directory.
