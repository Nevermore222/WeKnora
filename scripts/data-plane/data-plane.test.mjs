import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

function readScript(name) {
  return readFileSync(new URL(`./${name}`, import.meta.url), 'utf8')
}

const preflight = readScript('preflight.ps1')
const snapshot = readScript('snapshot.ps1')
const mergePlan = readScript('build-merge-plan.ps1')

test('preflight is read-only and records required database facts', () => {
  assert.match(preflight, /information_schema\.tables/)
  assert.match(preflight, /pg_constraint/)
  assert.match(preflight, /pg_extension/)
  assert.match(preflight, /Get-FileHash/)
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
  assert.match(mergePlan, /conflicts\.json/)
  assert.match(mergePlan, /unsupported-tables\.json/)
  assert.doesNotMatch(mergePlan, /DROP\s+DATABASE\s+Xelora/i)
})
