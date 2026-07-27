import { assert } from 'node:assert/strict'
import { test } from 'node:test'
import { filterExternalLinks, shouldShowExternalLink } from './externalLinks.ts'

test('hides GitHub and Tencent vendor links', () => {
  assert.equal(shouldShowExternalLink('https://github.com/Tencent/Xelora'), false)
  assert.equal(shouldShowExternalLink('https://console.cloud.tencent.com/cos'), false)
  assert.equal(shouldShowExternalLink('https://developers.weixin.qq.com/doc/foo'), false)
  assert.equal(shouldShowExternalLink('https://open.feishu.cn/app'), true)
})

test('filterExternalLinks removes hidden entries', () => {
  const links = filterExternalLinks([
    { label: 'GitHub', url: 'https://github.com/Tencent/Xelora' },
    { label: 'Feishu', url: 'https://open.feishu.cn/app' },
  ])
  assert.deepEqual(links, [{ label: 'Feishu', url: 'https://open.feishu.cn/app' }])
})
