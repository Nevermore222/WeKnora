import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const brandSource = readFileSync(new URL('./brandName.ts', import.meta.url), 'utf8')
const i18nSource = readFileSync(new URL('../i18n/index.ts', import.meta.url), 'utf8')

test('internal profile localizes generic product copy without rewriting provider or protocol names', () => {
  assert.match(brandSource, /export function localizeBrandText/)
  assert.match(brandSource, /brandProfileId !== ['"]internal['"]/)
  assert.match(brandSource, /Cloud/)
  assert.match(brandSource, /Xelora/)
  assert.match(i18nSource, /postTranslation/)
  assert.match(i18nSource, /localizeBrandText/)
})

test('internal localization consumes the complete lowercase runtime image labels', () => {
  assert.match(brandSource, /Xelora\(\?:-\(\?:app\|ui\)\)\?/)
  assert.match(brandSource, /\/gi/)
})
