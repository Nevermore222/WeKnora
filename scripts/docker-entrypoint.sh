#!/bin/bash
set -e

# Fix ownership of bind-mounted directories.
# When users bind-mount host directories (e.g. ./skills/preloaded),
# the mount inherits the host UID/GID which may differ from appuser.
# This entrypoint runs as root, fixes ownership, then drops privileges.

MOUNT_DIRS=(
    /app/skills/preloaded
    /data/files
)

for dir in "${MOUNT_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        chown -R appuser:appuser "$dir" 2>/dev/null || true
    fi
done

# Allow appuser to call the mounted Docker socket for controlled child
# sandboxes. The socket group id differs across Docker Desktop/Linux hosts, so
# create or reuse a matching group dynamically.
if [ -S /var/run/docker.sock ]; then
    DOCKER_SOCK_GID="$(stat -c '%g' /var/run/docker.sock)"
    if ! getent group "$DOCKER_SOCK_GID" >/dev/null 2>&1; then
        groupadd -g "$DOCKER_SOCK_GID" dockerhost 2>/dev/null || true
    fi
    DOCKER_SOCK_GROUP="$(getent group "$DOCKER_SOCK_GID" | cut -d: -f1)"
    if [ -n "$DOCKER_SOCK_GROUP" ]; then
        usermod -aG "$DOCKER_SOCK_GROUP" appuser 2>/dev/null || true
    fi
fi

# Merge built-in skills into preloaded.
# Built-in skills are backed up at /app/skills/_builtin during image build.
# After a bind-mount replaces /app/skills/preloaded, copy back any
# missing built-in skills without overwriting user-provided ones.
BUILTIN_DIR="/app/skills/_builtin"
PRELOADED_DIR="/app/skills/preloaded"

if [ -d "$BUILTIN_DIR" ]; then
    mkdir -p "$PRELOADED_DIR"
    for skill_dir in "$BUILTIN_DIR"/*/; do
        [ -d "$skill_dir" ] || continue
        skill_name="$(basename "$skill_dir")"
        if [ ! -d "$PRELOADED_DIR/$skill_name" ]; then
            cp -r "$skill_dir" "$PRELOADED_DIR/$skill_name"
        fi
    done
    chown -R appuser:appuser "$PRELOADED_DIR"
fi

exec gosu appuser "$@"
