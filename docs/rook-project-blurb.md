## Rook (Local Development Manager)

This project uses [Rook](https://github.com/andybarilla/rook) for local development orchestration. Rook manages container services (postgres, redis, caddy) and buildable services (api, worker) defined in `rook.yaml`.

**Key commands:**
- `rook up` — start all services (foreground with log streaming)
- `rook up --build` — rebuild containers from Dockerfiles before starting
- `rook up -d` — start detached
- `rook down` — stop all containers
- `rook status` — show service status
- `rook ports` — show allocated ports
- `rook ports --reset` — clear port allocations and stop containers

**How it works:**
- Rook allocates ports from a global pool (10000-60000) to avoid conflicts across projects
- Template vars in `rook.yaml` (`{{.Host.postgres}}`, `{{.Port.postgres}}`) resolve to container names and internal ports for container-to-container networking
- Template vars in mounted config files (like Caddyfile) are also resolved automatically
- All containers run on a shared `rook_<workspace>` network so they can reach each other by container name (`rook_<workspace>_<service>`)
- `env_file: .env` passes the project's `.env` file to containers via `--env-file`

**Files:**
- `rook.yaml` — workspace manifest (services, ports, profiles, dependencies)
- `.rook/scripts/` — devcontainer scripts copied during init (checked into git)
- `.rook/.cache/` — generated files (gitignored via `.rook/.gitignore`)
- `~/.config/rook/ports.json` — global port allocations
- `~/.config/rook/workspaces.json` — registered workspace registry
