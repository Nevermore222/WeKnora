import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./UserMenu.vue', import.meta.url), 'utf8')

test('keeps Claw Skill and Chrome extension entries hidden', () => {
  assert.match(source, /const showExternalResourceLinks = false/)

  const hiddenMenu = source.match(
    /<template v-if="showExternalResourceLinks">([\s\S]*?)<\/template>/,
  )?.[1] ?? ''

  assert.match(hiddenMenu, /common\.clawhubSkill/)
  assert.match(hiddenMenu, /common\.chromeExtension/)
})
