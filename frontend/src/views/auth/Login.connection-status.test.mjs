import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./Login.vue', import.meta.url), 'utf8')

test('connected enterprise status uses a transient top message instead of login-card layout', () => {
  assert.match(
    source,
    /v-if="isDefaultEnterpriseRequired\(\) && defaultEnterpriseConnectionState !== 'ready'"/,
  )
  assert.match(
    source,
    /MessagePlugin\.success\(\{\s*content:\s*t\('enterprise\.connectionStatus\.ready'\),\s*duration:\s*3000,\s*placement:\s*'top'/s,
  )
})
