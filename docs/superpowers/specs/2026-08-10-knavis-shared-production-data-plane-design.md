# Knavis Shared Production Data Plane Design

**Date:** 2026-08-10

## Decision

`192.168.9.177` becomes the single production data plane for the internal
Knavis deployment. It owns the production PostgreSQL/pgvector database, MinIO
object storage for knowledge-base files, and the central workspace file root.

The local workstation remains a development host. It may attach to the shared
production data plane only through an explicitly selected passive profile for
administrative verification. Its normal development profile uses isolated
state. The production server is the only active worker and the only host that
runs production database migrations, document parsing, embedding, indexing,
and scheduled background work.

This design preserves the existing Xelora knowledge-base retrieval and agent
behavior. It changes data placement and deployment governance only; it does
not replace the Go AgentEngine, Progressive RAG, tool ordering, Controlled
Docker, or workspace authorization contracts.

## Goals

- Merge the current local and `192.168.9.177` production data while retaining
  non-conflicting records from both sources.
- Resolve conflicting records deterministically with the local record as the
  winner.
- Provide one production source of truth for users, organizations, knowledge
  bases, documents, vectors, agents, permissions, and file artifacts.
- Keep local development possible without allowing experimental code or
  duplicate workers to affect LAN users.
- Make future Knavis releases repeatable, auditable, and reversible.
- Preserve Knavis and external Xelora as separately branded releases that use
  the same approved core behavior.

## Non-goals

- Do not merge Docker volume files, WSL VHDX files, or SQLite files directly.
- Do not make the local workstation a production failover node in this phase.
- Do not introduce multi-site database replication, high availability, or
  automatic cross-host failover.
- Do not replace existing RAG, agent, sandbox, Skill, or SSE execution logic.
- Do not migrate n8n, CL, COBOL, or unrelated local development artifacts.
- Do not store secrets, database dumps, object archives, or workspace backups
  in Git.

## Production Architecture

```text
LAN Knavis Clients
        |
        v
192.168.9.177
  Knavis frontend and API               Active production node
        |                                      |
        +-------------------+------------------+
                            |
        +-------------------+-----------------------+
        |                   |                       |
        v                   v                       v
 PostgreSQL + pgvector    MinIO                 Windows workspace share
 production metadata,    knowledge-base        mounted as /workspaces
 RAG index, users,       documents and         by the production containers
 agents, permissions     persisted artifacts

Local workstation
  - dev profile: local database, local object storage, local workspaces
  - shared/passive profile: same production PostgreSQL, MinIO, and workspace
    share; no migrations or active background work
```

The production PostgreSQL service remains on `192.168.9.177` in Docker-managed
persistent storage. It is not moved to a local Windows drive merely to change
a path. Object storage and workspace shares are separate persistence domains
because they have different access and consistency requirements.

## Data Ownership And Storage Boundaries

| Data class | Production owner | Access rule |
| --- | --- | --- |
| PostgreSQL and pgvector | `192.168.9.177` | API containers use it; direct database access is restricted to approved maintenance and passive-host traffic |
| Knowledge-base documents and durable artifacts | MinIO on `192.168.9.177` | The application accesses one configured bucket and object namespace |
| Bound workspace files | Windows share on `192.168.9.177` | Both Docker Desktop hosts mount the same root as `/workspaces`; workspace authorization remains enforced by Xelora |
| Parser temporary files and caches | Local to each active container host | Ephemeral; never treated as synchronized data |
| Local development data | Local workstation only | Never becomes production data without the approved merge procedure |

MinIO is the central store for knowledge-base document objects because it has
object semantics, checksums, and application-supported configuration. The
current workspace implementation is path-based, so it uses a controlled
Windows file share rather than pretending that a MinIO bucket is a POSIX
workspace. Both hosts must expose the same mounted container path and
read/write behavior before production cutover.

## Runtime Roles

### `192.168.9.177`: active production node

- Serves all LAN clients.
- Runs production database migrations exactly once per release.
- Owns document parsing, embedding, indexing, scheduled jobs, and any queue
  consumer that has external side effects.
- Is the only node allowed to apply production configuration changes during a
  release window.

### Local workstation: isolated development profile

- Uses a distinct Compose project and distinct PostgreSQL, object-storage,
  and workspace locations.
- Is the default profile for code, prompt, tool, and agent-runtime changes.
- May receive an approved, sanitized production snapshot for reproduction, but
  never points arbitrary in-progress code at the production database.

### Local workstation: shared/passive profile

- Uses the same production PostgreSQL, MinIO endpoint, and workspace share as
  `192.168.9.177` when a controlled verification session requires it.
- Does not execute database migrations, document parsing, embedding, indexing,
  scheduled jobs, or queue consumption.
- Is not published as a LAN client endpoint. Firewall and Compose settings
  keep it limited to local administrative access.

The current runtime configuration does not yet prove that all active workers
can be disabled independently. The implementation may not enable the
shared/passive profile until that behavior is verified in source and in a test
deployment. If an independent worker switch is unavailable, the local shared
profile is limited to stopped-container maintenance and direct health checks.

## Migration Rules

### Source snapshots

During the approved maintenance window, stop application and worker
containers on both hosts before taking snapshots. For each source, create:

- a PostgreSQL logical dump in a portable archive format, plus database role
  and extension metadata required for restore;
- a manifest of every knowledge-base file and durable artifact, including its
  application reference, path or object key, size, and SHA-256 hash;
- a manifest and snapshot of all workspace files; and
- the Compose configuration and image digest inventory used by that source.

The original database volumes and source files remain intact until migration
acceptance and rollback retention have both elapsed.

### Database reconciliation

The two logical dumps are restored into isolated temporary databases. A third
temporary merge database is built from those snapshots. The reconciliation
process works in foreign-key dependency order:

1. system settings, tenants, organizations, users, and identity records;
2. memberships, roles, user capabilities, model configuration, and resource
   sharing records;
3. knowledge bases, documents, parse metadata, chunks, vector/index metadata,
   and document-to-file references;
4. agents, prompts, tools, Skills, sessions, workspace bindings, jobs, and
   artifacts; and
5. audit, notification, and other dependent records.

For two records with the same logical primary key and different contents, the
local record wins. For a record present on only one source, it is retained.
Every conflict produces an auditable report containing table, key, source
hashes, selected source, and dependent-record action. A conflict that cannot
be reconciled without breaking a foreign-key or object reference blocks the
cutover; it is not silently dropped.

### File reconciliation

Files are reconciled by the database reference and content hash, not by a
blind directory copy. A file referenced by a retained local record wins when
the same logical object key has divergent content. The displaced server file
is copied into a dated quarantine area and recorded in the migration report.
Non-conflicting server files are preserved. The final database is checked to
ensure every retained file reference resolves to exactly one object or
workspace path in the central stores.

## Cutover Procedure

1. Complete preflight checks, source snapshots, and checksum validation.
2. Restore and reconcile data only in the temporary merge environment.
3. Run the acceptance checks against the merged temporary data.
4. Create a fresh backup of the untouched `192.168.9.177` production state.
5. Promote the verified merge database and reconciled files to the central
   production stores.
6. Configure the production application to use central PostgreSQL, MinIO, and
   the central workspace root, then start only the `192.168.9.177` active
   services.
7. Verify health, login, knowledge-base retrieval, agent configuration,
   document upload/download, and workspace artifact output.
8. Publish the internal client configuration whose default origin is
   `http://192.168.9.177:8080`, then keep the local machine off the LAN
   production path.

No destructive Docker volume command, raw volume replacement, or global prune
is part of this procedure.

## Acceptance Criteria

The production promotion may proceed only when all of the following pass:

- PostgreSQL schema versions and required extensions are compatible on both
  source snapshots.
- All merge conflicts have a report and the local-wins policy was applied.
- Row counts and key business entities reconcile against both source
  manifests.
- Foreign keys, document-to-object references, artifact references, and
  workspace bindings are valid.
- A sample from each retained knowledge base can be opened and retrieved.
- A representative user from each source can log in and see expected spaces,
  permissions, agents, and knowledge bases.
- Upload, parse, retrieve, agent chat, and a preloaded file-producing Skill
  succeed through the production node.
- The production app, frontend, PostgreSQL, object storage, and sandbox
  services are healthy after restart.
- Client login reaches `192.168.9.177` and exposes Knavis branding.

## Rollback

Rollback is a configuration and snapshot restoration, not a second merge:

1. Stop the newly promoted active application services.
2. Restore the pre-cutover `192.168.9.177` PostgreSQL snapshot, object-store
   snapshot, workspace snapshot, and Compose configuration.
3. Start the prior verified server images with `--no-build` semantics.
4. Verify the prior health, login, retrieval, and workspace checks.
5. Keep the failed merge artifacts and reports for diagnosis; do not overwrite
   either original source snapshot.

The client default remains or is restored to the prior verified server origin
only after the rollback health checks pass.

## Release And Synchronization Model

Production data does not require host-to-host synchronization after cutover;
both approved production-connected nodes read the same authoritative stores.
Code changes follow a controlled release path instead:

1. Develop and test in the isolated local `dev` profile.
2. Record the source commit, branch, test evidence, image digests, Compose
   configuration digest, and required migration list in a release manifest.
3. Build only services whose source or selected brand assets changed; reuse
   unchanged versioned images.
4. Apply database migration once on `192.168.9.177` after a backup and schema
   compatibility check.
5. Deploy the approved image set to `192.168.9.177`, validate production, and
   retain the previous image set for rollback.

Shared business changes are carried through explicit commits between the
Knavis internal branch and the external Xelora branch. Brand-specific assets,
product names, client defaults, and deployment metadata remain isolated.
No release is considered deployed merely because an image was built or a
Compose file was copied; container state, health endpoints, and user-facing
smoke checks are required.

## Security And Operations

- Use separate, host-specific secret files outside Git for database, MinIO,
  and share credentials.
- Restrict PostgreSQL, MinIO administration, and workspace-share access to
  the production host and explicitly approved maintenance clients on the LAN.
- Rotate credentials that have appeared in interactive troubleshooting before
  enabling the shared production data plane.
- Keep a scheduled `192.168.9.177` backup set consisting of PostgreSQL logical
  dumps, MinIO object manifests/snapshots, workspace-share snapshots, and
  restore-test evidence.
- Record deployment operator, time, source revisions, image digests, backup
  identifiers, migration report identifier, and acceptance results for every
  production cutover.

## Implementation Scope

The implementation plan following this design must first inspect the actual
database schema, storage adapter behavior, worker startup controls, and the
current `192.168.9.177` runtime state. It must then produce source-specific
migration tooling and a dry-run report before any remote production write.
It must not infer table mappings, share-mount support, credentials, or worker
switches from this document alone.
