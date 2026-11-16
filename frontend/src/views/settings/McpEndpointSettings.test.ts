import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'

const component = readFileSync(new URL('./McpEndpointSettings.vue', import.meta.url), 'utf8')
const localeFiles = ['zh-CN', 'en-US', 'ja-JP', 'vi-VN', 'ko-KR', 'ru-RU']

test('MCP endpoint settings includes the reusable usage tutorial', () => {
  assert.match(component, /mcp-endpoint-tutorial/)
  assert.match(component, /mcpEndpointSettings\.tutorial\.steps/)
  assert.match(component, /MCP_CONFIG_EXAMPLE/)
  assert.match(component, /knavis-knowledge/)
  assert.match(component, /your-knavis\.example\.com/)
})

test('all supported locales provide MCP tutorial translations', () => {
  for (const locale of localeFiles) {
    const source = readFileSync(new URL(`../../i18n/locales/${locale}.ts`, import.meta.url), 'utf8')
    assert.match(source, /["']?tutorial["']?\s*:/)
    assert.match(source, /["']?configExample["']?\s*:/)
    assert.doesNotMatch(source, /xelora-knowledge/)
    assert.doesNotMatch(source, /your-xelora\.example\.com/)
  }
})
