import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const localeUrls = {
  'en-US': new URL('./src/i18n/locales/en-US.ts', import.meta.url),
  'ja-JP': new URL('./src/i18n/locales/ja-JP.ts', import.meta.url),
  'vi-VN': new URL('./src/i18n/locales/vi-VN.ts', import.meta.url),
}

const leafEntries = (value, prefix = '', entries = new Map()) => {
  for (const [key, child] of Object.entries(value)) {
    const path = prefix ? `${prefix}.${key}` : key
    if (child && typeof child === 'object' && !Array.isArray(child)) {
      leafEntries(child, path, entries)
    } else {
      entries.set(path, child)
    }
  }
  return entries
}

const tokens = (value) =>
  [...String(value).matchAll(/\{[A-Za-z0-9_]+\}/g)].map(([token]) => token).sort()

for (const tag of ['ja-JP', 'vi-VN']) {
  test(`${tag} matches the English key and interpolation contract`, async () => {
    assert.ok(existsSync(localeUrls[tag]), `${tag} locale module must exist`)
    const [{ default: baselineLocale }, { default: candidateLocale }] = await Promise.all([
      import(localeUrls['en-US']),
      import(localeUrls[tag]),
    ])
    const baseline = leafEntries(baselineLocale)
    const candidate = leafEntries(candidateLocale)
    assert.deepEqual([...candidate.keys()].sort(), [...baseline.keys()].sort())
    for (const [key, source] of baseline) {
      assert.deepEqual(tokens(candidate.get(key)), tokens(source), `${tag}: ${key}`)
    }
  })
}

test('all UI locale selectors and registries expose Japanese and Vietnamese', () => {
  const files = [
    './src/i18n/index.ts',
    './src/i18n/embed.ts',
    './src/views/settings/GeneralSettings.vue',
    './src/views/auth/Login.vue',
    './src/components/AgentEmbedChannelPanel.vue',
  ].map((path) => readFileSync(new URL(path, import.meta.url), 'utf8'))

  for (const source of files) {
    assert.match(source, /ja-JP/, 'surface must include ja-JP')
    assert.match(source, /vi-VN/, 'surface must include vi-VN')
  }
})

test('embed locale normalization covers Japanese and Vietnamese prefixes', () => {
  const source = readFileSync(new URL('./src/i18n/embed.ts', import.meta.url), 'utf8')
  assert.match(source, /startsWith\('ja'\).*return 'ja-JP'/)
  assert.match(source, /startsWith\('vi'\).*return 'vi-VN'/)
  assert.match(source, /'ja-JP': deepMerge/)
  assert.match(source, /'vi-VN': deepMerge/)
})
