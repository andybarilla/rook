# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

Service to provide title abstractor capabilities via LLM including checking chain of title and liens.

## Architecture Overview

**Development Context**

- see @docs/ROADMAP.md for current status and next steps
- Tech stack: TBD
- Task-based development workflow with numbered tasks in `docs/tasks/` directory
- **Current Status**: Ready for planning
- **IMPORTANT**: all work should be done as TDD
- **Finding the next task to plan**: Scan the roadmap top-down. Skip checked items and items that already have a design doc in `docs/plans/`. The first unchecked item with no design doc is what to plan next. See `docs/ROADMAP.md` "Development Workflow" for full details.

## Git Workflow

- **Branch naming**: `task/XXX-short-description` (e.g., `task/001-database-schema`)
- **Always create a feature branch** before starting work on a task — never commit directly to `main`
- **One branch per task** — each task in `/docs/tasks` gets its own branch
- **Branch from `main`**: `git checkout -b task/XXX-description main`
- **Commit often** with clear messages describing what changed and why
- **Create a PR** when the task is complete and all tests pass
