import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

function readScript(name) {
  return readFileSync(new URL('./' + name, import.meta.url), 'utf8')
}

const packageMaterials = readScript('add-knavis-shared-data-plane-materials.ps1')
const migrationRunner = readScript('run-knavis-production-migration.ps1')
const passiveLauncher = readScript('start-knavis-passive.ps1')
const runbook = readFileSync(new URL('../docs/deployment/knavis-shared-production-data-plane.md', import.meta.url), 'utf8')

test('release materials include only templates and lifecycle tooling', () => {
  assert.match(packageMaterials, /docker-compose\.knavis-production\.yml/)
  assert.match(packageMaterials, /\.env\.knavis-production\.example/)
  assert.match(packageMaterials, /run-knavis-production-migration\.ps1/)
  assert.match(packageMaterials, /knavis-shared-production-data-plane\.md/)
  assert.doesNotMatch(packageMaterials, /merged-local-wins\.dump/)
  assert.doesNotMatch(packageMaterials, /data-plane\\.*\.zip/)
  assert.match(packageMaterials, /Join-Path \(Join-Path \$destinationRoot 'scripts\\data-plane'\) \$script\.Name/)
})

test('packaged passive launcher can use the enclosing deployment root', () => {
  assert.match(passiveLauncher, /DeploymentRoot/)
  assert.match(passiveLauncher, /Join-Path \$deploymentRoot "docker-compose\.yml"/)
  assert.match(runbook, /-DeploymentRoot \./)
})

test('runbook keeps logical merge approval and production restore separate', () => {
  assert.match(runbook, /local wins/i)
  assert.match(runbook, /no unattended production restore/i)
  assert.match(runbook, /knavis-candidate-/)
  assert.match(runbook, /do not use knavis-production for\s+candidate validation/i)
  assert.match(runbook, /-f shared-data-plane\\docker-compose\.knavis-production\.yml/)
  assert.match(runbook, /shared-data-plane\\scripts\\verify-shared-workspace\.ps1/)
  assert.match(runbook, /start-merge-candidate\.ps1/)
  assert.match(runbook, /run-knavis-production-migration\.ps1/)
})

test('production migration runner requires a verified backup and clean state', () => {
  assert.match(migrationRunner, /BackupManifest/)
  assert.match(migrationRunner, /BackupDump/)
  assert.match(migrationRunner, /schema_migrations/)
  assert.match(migrationRunner, /AUTO_RECOVER_DIRTY/)
  assert.match(migrationRunner, /--no-deps/)
  assert.doesNotMatch(migrationRunner, /docker\s+volume\s+rm/i)
})
