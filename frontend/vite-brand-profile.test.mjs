import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./vite.config.ts', import.meta.url), 'utf8')

test('Vite builds profile-specific document titles', () => {
  assert.match(source, /transformIndexHtml/)
  assert.match(source, /VITE_BRAND_PROFILE/)
  assert.match(source, /Knavis/)
  assert.match(source, /Xelora Embed/)
})
