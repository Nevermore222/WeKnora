import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

function readProjectFile(path) {
  return readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')
}

const production = readProjectFile('docker-compose.knavis-production.yml')
const passive = readProjectFile('docker-compose.knavis-passive.yml')
const productionEnv = readProjectFile('.env.knavis-production.example')
const passiveEnv = readProjectFile('.env.knavis-passive.example')
const launcher = readFileSync(new URL('./start-knavis-passive.ps1', import.meta.url), 'utf8')
const legacyWorkspaceSmoke = readFileSync(new URL('./host-workspace-mount-smoke.ps1', import.meta.url), 'utf8')

test('production profile centralizes MinIO and marks the app active', () => {
  assert.match(production, /XELORA_PROCESS_ROLE:\s*active/)
  assert.match(production, /STORAGE_TYPE:\s*minio/)
  assert.match(production, /MINIO_BUCKET_NAME/)
  assert.match(production, /KNAVIS_MAINTENANCE_BIND_IP/)
  assert.match(productionEnv, /^XELORA_PROCESS_ROLE=active$/m)
  assert.match(productionEnv, /^STORAGE_TYPE=minio$/m)
})

test('passive profile disables automatic migrations and binds locally only', () => {
  assert.match(passive, /XELORA_PROCESS_ROLE:\s*passive/)
  assert.match(passive, /AUTO_MIGRATE:\s*"false"/)
  assert.match(passive, /127\.0\.0\.1/)
  assert.match(passive, /container_name:\s*Knavis-passive-app/)
  assert.match(passive, /KNAVIS_PASSIVE_SERVICE_ENV_FILE/)
  assert.match(passiveEnv, /^XELORA_PROCESS_ROLE=passive$/m)
  assert.match(passiveEnv, /^AUTO_MIGRATE=false$/m)
  assert.match(launcher, /-p\s+\$projectName/)
  assert.match(launcher, /--no-deps\s+app\s+frontend/)
  assert.match(launcher, /KNAVIS_PASSIVE_SERVICE_ENV_FILE/)
  assert.doesNotMatch(launcher, /--build/)
  assert.match(legacyWorkspaceSmoke, /verify-shared-workspace\.ps1/)
  assert.doesNotMatch(legacyWorkspaceSmoke, /--build/)
})
