import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const loginSource = readFileSync(new URL('./Login.vue', import.meta.url), 'utf8')
const zhSource = readFileSync(new URL('../../i18n/locales/zh-CN.ts', import.meta.url), 'utf8')
const enSource = readFileSync(new URL('../../i18n/locales/en-US.ts', import.meta.url), 'utf8')

test('login branding is supplied by the active brand profile', () => {
  assert.match(loginSource, /form-brand__mark">\{\{ brandProfile\.displayName\.charAt\(0\) \}\}/)
  assert.match(loginSource, /form-brand__name">\{\{ brandProfile\.displayName \}\}/)
  assert.doesNotMatch(loginSource, /form-brand__name">Xelora<\/span>/)
  assert.match(zhSource, /firstTime:\s*"首次使用 \{brand\}？"/)
  assert.match(enSource, /firstTime:\s*'New to \{brand\}\?'/)
  assert.match(loginSource, /auth\.firstTime',\s*\{\s*brand:\s*brandProfile\.displayName\s*\}/)
})
