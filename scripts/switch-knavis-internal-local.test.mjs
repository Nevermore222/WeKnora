import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./switch-knavis-internal-local.ps1', import.meta.url), 'utf8')

test('local internal deployment uses the Compose postgres service', () => {
  assert.match(
    source,
    /Set-EnvValue -Path \$internalEnvFile -Name "DB_HOST" -Value "postgres"/,
  )
})

test('local internal deployment retries transient HTTP startup failures', () => {
  assert.match(source, /function Wait-HttpHealthy/)
  assert.match(source, /Start-Sleep -Seconds 2/)
  assert.match(source, /Wait-HttpHealthy -Url \$url/)
})
