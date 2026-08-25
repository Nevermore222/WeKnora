import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./package-knavis-internal.ps1', import.meta.url), 'utf8')
const verifySource = readFileSync(new URL('./verify-knavis-internal-package.ps1', import.meta.url), 'utf8')
const stopSource = readFileSync(new URL('./stop-knavis-internal.ps1', import.meta.url), 'utf8')
const deploySource = readFileSync(new URL('./deploy-knavis-internal.ps1', import.meta.url), 'utf8')
const verifyLauncher = readFileSync(new URL('./verify.cmd', import.meta.url), 'utf8')
const deployLauncher = readFileSync(new URL('./deploy.cmd', import.meta.url), 'utf8')
const stopLauncher = readFileSync(new URL('./stop.cmd', import.meta.url), 'utf8')

test('internal package includes manual verification and lifecycle materials', () => {
  assert.match(source, /verify-knavis-internal-package\.ps1/)
  assert.match(source, /stop-knavis-internal\.ps1/)
  assert.match(source, /scripts\\verify\.cmd/)
  assert.match(source, /scripts\\deploy\.cmd/)
  assert.match(source, /scripts\\stop\.cmd/)
  assert.match(source, /DEPLOYMENT-GUIDE\.zh-CN\.md/)
  assert.match(source, /SHA256SUMS\.txt/)
  assert.match(source, /client_sha256/)
  assert.match(source, /item\.Name -eq "data"/)
  assert.match(source, /\[System\.IO\.File\]::ReadAllText\(\$guideTemplatePath/)
})

test('internal package preserves public latest image tags', () => {
  assert.match(source, /public_latest_image_ids/)
  assert.match(source, /wechatopenai\/xelora-app:latest/)
  assert.match(source, /wechatopenai\/xelora-docreader:latest/)
  assert.match(source, /wechatopenai\/xelora-sandbox:latest/)
})

test('copied lifecycle scripts resolve their package directory when no argument is supplied', () => {
  assert.match(verifySource, /\[string\]\$PackageDir = ""/)
  assert.match(verifySource, /IsNullOrWhiteSpace\(\$PackageDir\)/)
  assert.match(stopSource, /\[string\]\$PackageDir = ""/)
  assert.match(stopSource, /IsNullOrWhiteSpace\(\$PackageDir\)/)
  assert.match(deploySource, /\[string\]\$PackageDir = ""/)
  assert.match(deploySource, /docker-compose\.yml/)
  assert.match(deploySource, /IsNullOrWhiteSpace\(\$PackageDir\)/)
})

test('Windows launchers bypass only the child PowerShell process policy', () => {
  assert.match(verifyLauncher, /-ExecutionPolicy Bypass/)
  assert.match(verifyLauncher, /-File "%~dp0verify\.ps1"/)
  assert.match(deployLauncher, /-ExecutionPolicy Bypass/)
  assert.match(deployLauncher, /-File "%~dp0deploy\.ps1"/)
  assert.match(stopLauncher, /-ExecutionPolicy Bypass/)
  assert.match(stopLauncher, /-File "%~dp0stop\.ps1"/)
})
