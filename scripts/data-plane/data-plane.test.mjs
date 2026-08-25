import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

function readScript(name) {
  return readFileSync(new URL(`./${name}`, import.meta.url), 'utf8')
}

const preflight = readScript('preflight.ps1')
const snapshot = readScript('snapshot.ps1')
const mergePlan = readScript('build-merge-plan.ps1')
const executeMerge = readScript('execute-merge.ps1')
const startCandidate = readScript('start-merge-candidate.ps1')
const importFiles = readScript('import-files-to-minio.ps1')
const verifyMerge = readScript('verify-merge.ps1')
const common = readScript('common.ps1')

test('preflight is read-only and records required database facts', () => {
  assert.match(preflight, /information_schema\.tables/)
  assert.match(preflight, /pg_constraint/)
  assert.match(preflight, /pg_extension/)
  assert.match(preflight, /Get-FileHash/)
  assert.match(preflight, /\$missingLocalFileReferences/)
  assert.match(preflight, /missing-local-file-references\.json/)
  assert.match(preflight, /Preflight blocked:/)
  assert.doesNotMatch(preflight, /DROP\s+DATABASE/i)
})

test('snapshot requires maintenance confirmation and uses logical archives', () => {
  assert.match(snapshot, /MaintenanceApproved/)
  assert.match(snapshot, /pg_dump/)
  assert.match(snapshot, /--format=custom/)
  assert.match(snapshot, /Get-FileHash/)
  assert.doesNotMatch(snapshot, /docker\s+volume\s+rm/i)
})

test('merge planning restores only disposable databases and reports collisions', () => {
  assert.match(mergePlan, /knavis_merge_/)
  assert.match(mergePlan, /pg_restore/)
  assert.match(mergePlan, /postgres_fdw/)
  assert.match(mergePlan, /--tmpfs/)
  assert.match(mergePlan, /\/var\/lib\/postgresql\/data:rw/)
  assert.match(mergePlan, /conflicts\.json/)
  assert.match(mergePlan, /unsupported-tables\.json/)
  assert.match(mergePlan, /merged-data-files\.tar\.gz/)
  assert.match(mergePlan, /merged-file-references\.json/)
  assert.doesNotMatch(mergePlan, /DROP\s+DATABASE\s+Xelora/i)
})

test('execution requires explicit local-wins approval and emits a portable archive', () => {
  assert.match(executeMerge, /ApproveLocalWins/)
  assert.match(executeMerge, /merge-plan\.sha256/)
  assert.match(executeMerge, /pg_dump/)
  assert.doesNotMatch(executeMerge, /docker\s+volume\s+rm/i)
})

test('candidate restore remains isolated from production databases', () => {
  assert.match(startCandidate, /knavis_merge_candidate_/)
  assert.match(startCandidate, /--tmpfs/)
  assert.match(startCandidate, /\/var\/lib\/postgresql\/data:rw/)
  assert.match(startCandidate, /pg_restore/)
  assert.match(startCandidate, /CREATE EXTENSION/)
  assert.doesNotMatch(startCandidate, /docker\s+volume\s+rm/i)
})

test('file migration preserves content hashes and only rewrites reviewed knowledge paths', () => {
  assert.match(importFiles, /local:\/\//)
  assert.match(importFiles, /minio:\/\//)
  assert.match(importFiles, /file-references\.json/)
  assert.match(importFiles, /Get-FileHash/)
  assert.match(verifyMerge, /foreign_key_check/)
  assert.match(verifyMerge, /object-manifest/)
})

test('data-files volume scanning transports nested shell safely from Windows', () => {
  assert.match(common, /\[Convert\]::ToBase64String/)
  assert.match(common, /base64 -d \| sh/)
  assert.doesNotMatch(common, /'alpine:3\.20', 'sh', '-c', \$script/)
  assert.doesNotMatch(common, /printf '\\t'/)
})
