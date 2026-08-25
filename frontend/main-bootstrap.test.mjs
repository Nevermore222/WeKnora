import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./src/main.ts', import.meta.url), 'utf8')

test('desktop enterprise bootstrap does not block the initial Vue mount', () => {
  assert.match(source, /void\s+initDefaultEnterpriseProfile\(\)/)
  assert.doesNotMatch(source, /await\s+initDefaultEnterpriseProfile\(\)/)
  assert.match(source, /router\.isReady\(\)\.finally\(\(\)\s*=>\s*\{/)
})
