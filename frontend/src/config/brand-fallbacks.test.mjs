import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const bootstrapSource = readFileSync(new URL('../utils/default-enterprise-bootstrap.ts', import.meta.url), 'utf8')
const desktopRemoteSource = readFileSync(new URL('../api/desktop-remote/index.ts', import.meta.url), 'utf8')

test('enterprise server name fallbacks use the active brand profile', () => {
  assert.match(bootstrapSource, /brandDisplayName/)
  assert.match(desktopRemoteSource, /brandDisplayName/)
  assert.doesNotMatch(bootstrapSource, /\|\| ['"]Xelora Server['"]|\?\? ['"]Xelora Server['"]|Xelora Server/)
  assert.doesNotMatch(desktopRemoteSource, /\|\| ['"]Xelora Server['"]|\?\? ['"]Xelora Server['"]|Xelora Server/)
})
