import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const scriptPath = new URL('./publish-knavis-client-release.ps1', import.meta.url)

test('Knavis release publisher maintains immutable versioned channels', () => {
  assert.ok(existsSync(scriptPath), 'release publisher must exist')
  const source = readFileSync(scriptPath, 'utf8')

  assert.match(source, /\[ValidateSet\("stable", "beta"\)\]/)
  assert.match(source, /Release version already exists/)
  assert.match(source, /Get-FileHash.*SHA256/)
  assert.match(source, /Join-Path \$channelsDirectory "\$Channel\.json"/)
  assert.match(source, /release-index\.json/)
  assert.match(source, /latest\.json/)
})
