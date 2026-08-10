# Knavis Shared Production Data Plane Runbook

## Purpose and boundaries

This runbook prepares a Knavis internal release in which 192.168.9.177 is the
only active production node. It owns PostgreSQL, Redis/Asynq, MinIO, and the
Windows workspace share. The local machine remains a development machine and
can use the passive profile only for controlled, loopback-only verification.

The Knavis and Xelora variants reuse the same approved application images and
core behavior. This package adds internal branding and operational profiles; it
does not replace or rebuild the existing Xelora image set.

The merge rule is local wins: when an identical primary-key row differs, the
local record replaces the server record. Server-only rows are retained. This
rule applies only after the generated reports have been reviewed. Do not merge
Docker volumes, WSL VHDX files, or raw PostgreSQL directories.

There is no unattended production restore in this release. The tooling creates
read-only inventories, logical snapshots, a reviewed merge archive, an isolated
candidate database, and validation evidence. A production restore remains a
separate, explicitly approved maintenance operation.

## Safety gates

1. Keep all secrets in private environment files. Do not commit, package, or
   paste database, MinIO, SMB, or encryption credentials into a command line.
2. Do not set AUTO_RECOVER_DIRTY=true. It defeats the migration safety gate.
3. Stop all active Knavis/Xelora application containers before snapshot or
   migration work. Do not use Compose down with volumes.
4. Do not start the local passive profile before the production service,
   firewall rules, SMB share, and least-privilege credentials are verified.
5. Do not direct any client to 192.168.9.177 until candidate validation and
   the production application smoke test have been recorded.
6. Treat every artifact directory, SHA256SUMS.txt, conflict report, and
   verification report as change-control evidence.

## Build the release supplement

Use a clean source checkout that contains the approved internal Knavis commit.
First run the checked source tests:

    node --test scripts\shared-data-plane-compose.test.mjs
    node --test scripts\data-plane\data-plane.test.mjs
    node --test scripts\shared-data-plane-release.test.mjs

Build the normal Knavis server package using the existing package workflow. Do
not alter or replace the existing package script in a dirty source checkout.
Then add only the non-secret shared-data-plane supplement to the staged server
package:

    .\scripts\add-knavis-shared-data-plane-materials.ps1 -ServerPackageDir D:\Knavis\release-staging\server

The helper requires the base docker-compose.yml in the target package and
creates a shared-data-plane directory. It copies templates, lifecycle scripts,
the data-plane tools, and this runbook. It rejects .env files, database dumps,
data archives, SQLite files, and data directories, and writes a SHA256 manifest.
Zip the complete staged package only after reviewing that manifest.

## Prepare 192.168.9.177

1. Extract the base Knavis package and its shared-data-plane supplement to a
   server-owned directory, for example D:\Knavis\current. Do not overwrite a
   known-good Xelora package; use a separate Knavis release directory.
2. Copy shared-data-plane\.env.knavis-production.example to a private .env in
   the deployment root. Replace all placeholders with the approved values.
3. Preserve existing TENANT_AES_KEY, SYSTEM_AES_KEY, CRYPTO_MASTER_KEY, and
   CRYPTO_SALT values whenever importing records that already use them. A new
   encryption key makes imported secrets unreadable.
4. Keep AUTO_MIGRATE=false and AUTO_RECOVER_DIRTY=false. The one-time
   migration runner performs migrations after a verified backup.
5. Point XELORA_WORKSPACE_HOST_ROOT to D:\Knavis\workspaces and create that
   directory on the server. Share it with only the Knavis service identity and
   the local development administrator. Map the share on the local machine to
   the private path configured by .env.knavis-passive.
6. Bind PostgreSQL, Redis, and MinIO to the server maintenance IP only. Add
   Windows Firewall rules which allow ports 5432, 6379, 9000, and 9001 only
   from the approved local development IP. Do not expose these maintenance
   ports to the general LAN.
7. Confirm the active production Compose configuration without starting it:

    docker compose -p knavis-production --env-file .env -f docker-compose.yml -f shared-data-plane\docker-compose.knavis-production.yml --profile minio config --quiet

The private .env must reference already loaded or approved Knavis images.
Cutover uses Compose up with --no-build so the production image identity stays
explicit.

## Establish source inventories

Run a read-only preflight on the current server and local source. Use actual
container and volume names from each host. The output root must be a protected
artifact directory with enough space for logical dumps and file archives.

On 192.168.9.177:

    .\shared-data-plane\scripts\data-plane\preflight.ps1 -SourceName server -PostgresContainer <server-postgres-container> -DatabaseName Xelora -DatabaseUser <database-user> -DataFilesVolume <server-data-files-volume> -WorkspaceRoot D:\Knavis\workspaces -OutputRoot D:\Knavis\migration-artifacts

On the local development host:

    .\scripts\data-plane\preflight.ps1 -SourceName local -PostgresContainer <local-postgres-container> -DatabaseName Xelora -DatabaseUser <database-user> -DataFilesVolume <local-data-files-volume> -WorkspaceRoot <local-workspace-root> -OutputRoot D:\Knavis\migration-artifacts

Review both preflight.ok markers, database-schema.json files, migration state,
file-references.json files, data-files-manifest.json files, and workspace
manifests. Stop if an extension, migration, schema, or storage reference differs
unexpectedly.

## Freeze and take logical snapshots

Schedule a maintenance window. Stop all application containers on both nodes
while leaving the PostgreSQL containers available. Do not remove volumes. Then
run snapshot.ps1 against each source, passing the matching preflight directory
and -MaintenanceApproved. For example:

    .\shared-data-plane\scripts\data-plane\snapshot.ps1 -SourceName server -PostgresContainer <server-postgres-container> -DatabaseName Xelora -DatabaseUser <database-user> -DataFilesVolume <server-data-files-volume> -WorkspaceRoot D:\Knavis\workspaces -PreflightDir <server-preflight-artifact> -OutputRoot D:\Knavis\migration-artifacts -MaintenanceApproved

Repeat with SourceName local and the local source values. Snapshot output is a
logical PostgreSQL custom dump plus data-file and workspace manifests. Verify
each generated SHA256SUMS.txt before moving artifacts to the reconciliation
host.

## Build and review the merge plan

Run this only on an administrative machine with Docker and no active production
restore target:

    .\scripts\data-plane\build-merge-plan.ps1 -ServerSnapshotDir <server-snapshot-artifact> -LocalSnapshotDir <local-snapshot-artifact> -OutputRoot D:\Knavis\migration-artifacts

Review the generated merge-plan.json, merge-plan.sha256, merge.sql,
conflicts.json, unsupported-tables.json, table-order.json, merged file
references, and all SHA256 values. Confirm that local wins is acceptable for
every listed collision, that server-only records are retained, and that no
unsupported table needs a manual migration procedure.

Execute the reviewed logical merge only after recording the SHA-256 of
merge.sql:

    .\scripts\data-plane\execute-merge.ps1 -MergePlanDir <merge-plan-artifact> -ReviewedPlanHash <merge-sql-sha256> -OutputRoot D:\Knavis\migration-artifacts -ApproveLocalWins

This writes a temporary-candidate archive. It does not restore 192.168.9.177.

## Restore and validate an isolated candidate

Start a disposable PostgreSQL candidate from the merged archive:

    .\scripts\data-plane\start-merge-candidate.ps1 -MergeExecutionDir <merge-execution-artifact> -OutputRoot D:\Knavis\migration-artifacts

The candidate has no published ports and is named with the
knavis_merge_candidate_ prefix. Keep it running only for validation. Record
candidate.json, then use its container and database name for file reference
updates.

Create a private MinIO credentials file with only the required access key and
secret. Do not include it in any artifact directory. Use a dedicated candidate
bucket named knavis-candidate-<merge-id>; do not use knavis-production for
candidate validation. Prefer a disposable MinIO endpoint. If the server MinIO
endpoint must be used, keep this candidate bucket distinct and restrict its
credentials to that bucket. Import the reviewed merged file archive and apply
reference updates only to the candidate database:

    .\scripts\data-plane\import-files-to-minio.ps1 -DataFilesArchive <merge-plan-artifact>\merged-data-files.tar.gz -DataFilesManifest <merge-plan-artifact>\merged-data-files-manifest.json -FileReferencesReport <merge-plan-artifact>\merged-file-references.json -ReviewedReferencesHash <reviewed-reference-report-sha256> -MinioCredentialEnvFile <private-minio-env-file> -MinioEndpoint http://<candidate-minio-endpoint> -TargetBucket knavis-candidate-<merge-id> -PostgresContainer <candidate-container> -DatabaseName <candidate-database> -DatabaseUser postgres -OutputRoot D:\Knavis\migration-artifacts -ApproveFileImport -ApplyReviewedReferences

Verify candidate rows, constraints, object hashes, and workspace manifests:

    .\scripts\data-plane\verify-merge.ps1 -PostgresContainer <candidate-container> -DatabaseName <candidate-database> -DatabaseUser postgres -ObjectManifest <file-import-artifact>\object-manifest.json -MinioCredentialEnvFile <private-minio-env-file> -MinioEndpoint http://<candidate-minio-endpoint> -TargetBucket knavis-candidate-<merge-id> -MergePlanDir <merge-plan-artifact> -ExpectedWorkspaceManifest <approved-workspace-manifest> -TargetWorkspaceRoot D:\Knavis\workspaces -OutputRoot D:\Knavis\migration-artifacts

Stop at this checkpoint. Review verification.json and all hashes before any
production restore. The candidate container can be removed manually after the
evidence is retained.

## Controlled production promotion

Production promotion requires a separate approval after the candidate evidence
is reviewed. Take and hash a fresh pre-promotion production backup. Restore the
approved logical archive through a documented, operator-reviewed PostgreSQL
maintenance procedure. Do not use raw volume copy, destructive Docker commands,
or an unreviewed SQL file.

After the production database restore is separately approved, run the reviewed
file import once more against the production database and knavis-production
bucket. This is the first production object/reference write:

    .\shared-data-plane\scripts\data-plane\import-files-to-minio.ps1 -DataFilesArchive <merge-plan-artifact>\merged-data-files.tar.gz -DataFilesManifest <merge-plan-artifact>\merged-data-files-manifest.json -FileReferencesReport <merge-plan-artifact>\merged-file-references.json -ReviewedReferencesHash <reviewed-reference-report-sha256> -MinioCredentialEnvFile <private-production-minio-env-file> -MinioEndpoint http://<production-minio-endpoint> -TargetBucket knavis-production -PostgresContainer <production-postgres-container> -DatabaseName Xelora -DatabaseUser <database-user> -OutputRoot D:\Knavis\migration-artifacts -ApproveFileImport -ApplyReviewedReferences

After the restore, run the one-time migration runner with the recorded backup:

    .\shared-data-plane\scripts\run-knavis-production-migration.ps1 -EnvFile .env -BaseComposeFile .\docker-compose.yml -ProductionComposeFile .\shared-data-plane\docker-compose.knavis-production.yml -BackupManifest <pre-promotion-SHA256SUMS.txt> -BackupDump <pre-promotion-database.dump> -PostgresContainer <production-postgres-container> -DatabaseName Xelora -DatabaseUser <database-user> -OutputRoot D:\Knavis\migration-artifacts -MaintenanceApproved

The runner refuses dirty schema_migrations rows, requires all application
containers to be stopped, validates the approved backup hash, runs only the
versioned migration command, and writes a migration report.

Start the internal production node without a build:

    docker compose -p knavis-production --env-file .env -f docker-compose.yml -f shared-data-plane\docker-compose.knavis-production.yml --profile minio up -d --no-build

Validate the API, login, knowledge-base retrieval, agent streaming, tool use,
workspace authorization, and a skill-produced workspace artifact. Only after
these checks pass should the internally branded client default endpoint be
published for LAN users.

## Local development after cutover

Keep ordinary local development on its isolated local Compose/database
configuration. Do not point a developer .env at production by default. For
controlled shared-data-plane verification only, create a private
.env.knavis-passive in the deployment root from
shared-data-plane\.env.knavis-passive.example and start:

    .\shared-data-plane\scripts\start-knavis-passive.ps1 -DeploymentRoot . -EnvFile .env.knavis-passive

The passive profile binds app and frontend ports only to 127.0.0.1, does not
run migrations, and does not start workers, schedulers, housekeeping, audit
retention, or IM consumers. Verify the SMB mount before testing workspace
behavior:

    .\shared-data-plane\scripts\verify-shared-workspace.ps1 -HostRoot K:\KnavisWorkspaces -ContainerName Knavis-passive-app

## Rollback

Rollback is backup-based. Keep the pre-promotion logical database dump, object
manifest, workspace manifest, image identifiers, private environment file
checksum, and verified production migration report. If production validation
fails, stop the active application containers, restore the recorded
pre-promotion logical backup using the approved maintenance procedure, restore
the matching object and workspace state, and start the previously approved
Knavis images with --no-build.

Do not attempt a reverse merge. Record the failed verification evidence and
resolve the underlying discrepancy in an isolated candidate before scheduling
another maintenance window.
