import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

function readScript(name) {
  return readFileSync(new URL(`./${name}`, import.meta.url), 'utf8')
}

const preflight = readScript('preflight.ps1')
const snapshot = readScript('snapshot.ps1')

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
