# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

Rook — a cross-platform, open-source local development environment manager. Inspired by Laravel Herd, extensible via plugins to any language stack.

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

## Post-Merge Checklist

After a task's PR is merged:
- **Update `docs/ROADMAP.md`**: Mark the completed item with `[x]` and add a task file reference (e.g., `— See: docs/tasks/004-caddy-manager.md`)
- Commit the roadmap update to the working branch (e.g., `agent-1`) and push
