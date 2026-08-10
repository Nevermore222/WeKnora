# Knavis Minimal Client Distribution Design

## Goal

Make the tested local Knavis desktop client connect reliably to the already
running Knavis service on 192.168.9.177. Client delivery is independent from
server delivery.

## Scope

- Compare the local client release with the client package stored under the
  existing Knavis release directory on 192.168.9.177.
- Upload and publish only the client package when its version or file hashes
  differ.
- Verify the remote service health and the client-to-server API connection.

## Non-Goals

- Do not rebuild, replace, stop, or otherwise redeploy the server.
- Do not change Docker Compose, images, containers, volumes, PostgreSQL,
  Redis, MinIO, workspaces, or server environment files.
- Do not merge local and remote databases or introduce an update platform,
  static download service, or client auto-update.

## Release Boundary

The existing Knavis deployment at
D:\Knavis\knavis-internal-0.6.2-c5f40aeb is the production baseline.
The client package is the only mutable deployment artifact in this workflow.
The existing client subdirectory remains the manual distribution location.

## Workflow

1. Read the local client release manifest, executable version, and SHA256
   values without rebuilding it.
2. Read the remote client manifest and SHA256 values over authenticated SSH.
3. Read the remote service health and capability endpoint. Record the result
   without changing any server resource.
4. If client artifacts match, make no remote file change.
5. If they differ, upload the local client release to a new temporary
   directory below the existing Knavis release directory, verify all hashes,
   then replace only the client directory with the verified package while
   keeping the previous client directory as rollback material.
6. Start the uploaded client locally and verify it reaches the remote Knavis
   service. A failed client check restores the previous client directory only.

## Preconditions

- The SSH host key for 192.168.9.177 is verified.
- A deployment-only SSH key is authorized on 192.168.9.177 before remote
  comparison or upload.
- The local client release directory is explicitly identified. Dirty source
  files are never treated as a release artifact.

## Safety Rules

- Every remote operation is either read-only or restricted to the existing
  Knavis client directory.
- No command may invoke Docker, Docker Compose, database tooling, volume
  tooling, or a server deployment script on 192.168.9.177.
- A client mismatch never triggers a server update.
- A failed upload or hash check leaves the current remote client package
  untouched.

## Acceptance Criteria

- Remote Knavis containers and data remain unchanged.
- The published client package has the same version and SHA256 values as the
  selected local release.
- The client opens, uses the Knavis brand, and can reach the remote login and
  capability API.
- When no artifact difference exists, the workflow reports that no upload was
  performed.
