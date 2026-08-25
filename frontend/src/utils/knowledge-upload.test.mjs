import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const {
  clearFormDataContentType,
  formatKnowledgeUploadError,
  isKnowledgeFileUploadComplete,
  getUploadFileDiagnostics,
  snapshotUploadFiles,
} = await import('./knowledgeUpload.ts')

const here = dirname(fileURLToPath(import.meta.url))
const requestSource = readFileSync(resolve(here, 'request.ts'), 'utf8')
const knowledgeViewSource = readFileSync(
  resolve(here, '../views/knowledge/KnowledgeBase.vue'),
  'utf8',
)
const uploadSourceDropdown = readFileSync(
  resolve(here, '../views/knowledge/components/KbUploadSourceDropdown.vue'),
  'utf8',
)

function postUploadSource() {
  const start = requestSource.indexOf('export function postUpload')
  const end = requestSource.indexOf('export function postChat', start)
  assert.notEqual(start, -1, 'postUpload should exist')
  assert.notEqual(end, -1, 'postUpload should be bounded by postChat')
  return requestSource.slice(start, end)
}

test('multipart upload lets the browser generate the boundary', () => {
  const source = postUploadSource()

  assert.doesNotMatch(
    source,
    /["']Content-Type["']\s*:/,
    'FormData uploads must not override the browser-generated multipart boundary',
  )
  assert.match(source, /["']X-Request-ID["']\s*:/)
})

test('form-data uploads remove the default JSON content type', () => {
  const headers = {
    'Content-Type': 'application/json',
    'content-type': 'application/json',
    'X-Request-ID': 'request-1',
  }

  clearFormDataContentType(headers)

  assert.equal(headers['Content-Type'], undefined)
  assert.equal(headers['content-type'], undefined)
  assert.equal(headers['X-Request-ID'], 'request-1')
})

test('upload errors include server parsing details', () => {
  assert.equal(
    formatKnowledgeUploadError({
      message: 'File upload failed',
      details: 'multipart: NextPart: EOF',
    }, 'Upload failed'),
    'File upload failed: multipart: NextPart: EOF',
  )
  assert.equal(
    formatKnowledgeUploadError({ message: 'File upload failed' }, 'Upload failed'),
    'File upload failed',
  )
})

test('file upload success follows the server success contract', () => {
  assert.equal(
    isKnowledgeFileUploadComplete({ success: true, data: { file_size: 1536 } }),
    true,
  )
  assert.equal(
    isKnowledgeFileUploadComplete({ success: true, data: { file_size: 0 } }),
    true,
  )
  assert.equal(
    isKnowledgeFileUploadComplete({ success: true, data: {} }),
    true,
  )
  assert.equal(
    isKnowledgeFileUploadComplete({ success: false, data: { file_size: 1536 } }),
    false,
  )
  assert.match(knowledgeViewSource, /isKnowledgeFileUploadComplete\(responseData\)/)
})

test('upload diagnostics expose the browser file size and metadata', () => {
  const file = new File(['pdf-bytes'], 'test.pdf', { type: 'application/pdf' })
  Object.defineProperty(file, 'webkitRelativePath', { value: 'folder/test.pdf' })

  assert.deepEqual(getUploadFileDiagnostics(file), {
    name: 'test.pdf',
    size: 9,
    type: 'application/pdf',
    webkitRelativePath: 'folder/test.pdf',
  })
  assert.deepEqual(getUploadFileDiagnostics(null), {
    name: '',
    size: 0,
    type: '',
    webkitRelativePath: '',
  })
})

test('file selection preserves the WebView file handle until upload starts', () => {
  const handlerStart = uploadSourceDropdown.indexOf('const handleFilesChange')
  const handlerEnd = uploadSourceDropdown.indexOf('const handleUrlDialogConfirm', handlerStart)
  const handler = uploadSourceDropdown.slice(handlerStart, handlerEnd)

  assert.doesNotMatch(
    handler,
    /emit\('files', result\.validFiles\)\s*\n\s*input\.value = ''/,
  )
  assert.match(uploadSourceDropdown, /input\.value = ''\s*\n\s*input\.click\(\)/)
})

test('file selection snapshots bytes before the WebView file handle expires', async () => {
  const source = new File(['%PDF-1.4\nTest'], 'test.pdf', {
    type: 'application/pdf',
    lastModified: 123,
  })
  Object.defineProperty(source, 'webkitRelativePath', { value: 'folder/test.pdf' })

  const [snapshot] = await snapshotUploadFiles([source])

  assert.notEqual(snapshot, source)
  assert.equal(snapshot.name, 'test.pdf')
  assert.equal(snapshot.type, 'application/pdf')
  assert.equal(snapshot.lastModified, 123)
  assert.equal(snapshot.webkitRelativePath, 'folder/test.pdf')
  assert.equal(await snapshot.text(), '%PDF-1.4\nTest')
})
