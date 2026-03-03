# Flock Roadmap

An application to 

## Development Workflow

1. **Task Planning**

- Study the existing codebase and understand the current state
- Update `ROADMAP.md` to include the new task
- Priority tasks should be inserted after the last completed task
- **Finding the next task to plan:** Scan the roadmap top-down. Skip completed items (checked). Skip items that already have a design doc in `docs/plans/` — those are already planned. The first unchecked item with no design doc is the next thing to plan.

2. **Task Creation**

- Study the existing codebase and understand the current state
- Create a new task file in the `docs/tasks/` directory
- Name format: `XXX-description.md` (e.g., `001-db.md`)
- Include high-level specifications, relevant files, acceptance criteria, and implementation steps
- Refer to last completed task in the `docs/tasks/` directory for examples. For example, if the current task is `012`, refer to `011` and `010` for examples.
- Note that these examples are completed tasks, so the content reflects the final state of completed tasks (checked boxes and summary of changes). For the new task, the document should contain empty boxes and no summary of changes. Refer to `000-sample.md` as the sample for initial state.

3. **Task Implementation**

- Follow the specifications in the task file
- Implement features and functionality
- Update step progress within the task file after each step
- Stop after completing each step and wait for further instructions

4. **Roadmap Updates**

- Mark completed tasks with ✅ in the roadmap
- Add reference to the task file (e.g., `See: docs/tasks/001-db.md`)

## Development Phases

