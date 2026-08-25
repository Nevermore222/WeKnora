import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const pageSource = readFileSync(new URL('./ClientDownload.vue', import.meta.url), 'utf8')
const routerSource = readFileSync(new URL('../../router/index.ts', import.meta.url), 'utf8')
const nginxSource = readFileSync(new URL('../../../nginx.conf', import.meta.url), 'utf8')
const composeSource = readFileSync(new URL('../../../../docker-compose.knavis-internal.yml', import.meta.url), 'utf8')

test('download page is public and uses the release manifest for both actions', () => {
  assert.match(routerSource, /path:\s*['"]\/download['"]/)
  assert.match(routerSource, /name:\s*['"]clientDownload['"]/) 
  assert.match(routerSource, /requiresAuth:\s*false,\s*requiresInit:\s*false/)
  assert.match(pageSource, /\/downloads\/channels\/stable\.json/)
  assert.match(pageSource, /\/downloads\/latest\.json/)
  assert.match(pageSource, /parseChannelManifest/)
  assert.match(pageSource, /Invalid release SHA-256/)
  assert.match(pageSource, /releaseHistory/)
  assert.match(pageSource, /downloadLatest/)
  assert.match(pageSource, /downloadHref/)
  assert.match(pageSource, /更新客户端/)
  assert.match(pageSource, /下载客户端/)
})

test('Nginx serves release metadata and ZIP files without SPA fallback', () => {
  const manifestLocation = nginxSource.indexOf('location = /downloads/latest.json')
  const zipLocation = nginxSource.indexOf('location ~* ^/downloads/')
  const spaLocation = nginxSource.indexOf('location / {')

  assert.ok(manifestLocation >= 0)
  assert.ok(nginxSource.includes('location = /downloads/release-index.json'))
  assert.ok(nginxSource.includes('/downloads/channels/(stable|beta)'))
  assert.ok(zipLocation > manifestLocation)
  assert.ok(spaLocation > zipLocation)
  assert.match(nginxSource, /Content-Disposition "attachment"/)
  assert.match(nginxSource, /try_files \$uri =404/)
})

test('internal compose mounts the release directory read-only', () => {
  assert.match(
    composeSource,
    /- \$\{CLIENT_RELEASES_HOST_PATH:-\.\/dist\/client-releases\}:\/usr\/share\/nginx\/html\/downloads:ro/,
  )
})

test('download actions remain inside a narrow viewport', () => {
  assert.match(pageSource, /grid-template-columns:\s*minmax\(0,\s*1fr\)/)
  assert.match(
    pageSource,
    /\.release-console\s*\{[\s\S]*?width:\s*100%;[\s\S]*?box-sizing:\s*border-box;/,
  )
  assert.match(
    pageSource,
    /@media \(max-width: 820px\)\s*\{[\s\S]*?\.release-console\s*\{[\s\S]*?animation:\s*none;/,
  )
})
