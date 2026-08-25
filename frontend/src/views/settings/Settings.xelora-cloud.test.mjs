import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const settingsSource = readFileSync(new URL('./Settings.vue', import.meta.url), 'utf8')
const brandSource = readFileSync(new URL('../../config/brandProfile.ts', import.meta.url), 'utf8')

test('internal branding hides the Xelora Cloud settings surface', () => {
  assert.match(settingsSource, /const showXeloraCloudSettings = brandProfile\.showXeloraCloudSettings/)
  assert.match(settingsSource, /showXeloraCloudSettings && currentSection === 'xeloracloud'/)
  assert.match(settingsSource, /showXeloraCloudSettings\s*\?\s*\[\{ key: 'xeloracloud'/)
  assert.match(brandSource, /showXeloraCloudSettings: false/)
  assert.match(brandSource, /showXeloraCloudSettings: true/)
})
