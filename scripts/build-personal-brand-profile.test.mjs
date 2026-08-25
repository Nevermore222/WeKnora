import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./build-personal.ps1', import.meta.url), 'utf8')

test('desktop build passes the selected brand profile and uses a profile-specific executable name', () => {
  assert.match(source, /internal\/branding\.PackagedProfile=\$BrandProfile/)
  assert.match(source, /Knavis Personal/)
  assert.match(source, /"-o", "\$ApplicationName\.exe"/)
})

test('internal desktop releases inject an independent Knavis version and update channel', () => {
  assert.match(source, /\[string\]\$ReleaseVersion = ""/)
  assert.match(source, /\[string\]\$DefaultUpdateFeedURL = ""/)
  assert.match(source, /\$Version = "1\.0\.0"/)
  assert.match(source, /main\.packagedDesktopUpdateFeedURL=\$NormalizedDefaultUpdateFeedURL/)
  assert.match(source, /VITE_FRONTEND_VERSION/)
})
