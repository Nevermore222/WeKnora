interface KnowledgeUploadResponse {
  success?: boolean
  error?: unknown
}

export function clearFormDataContentType(headers: Record<string, unknown> | undefined) {
  if (!headers) return

  delete headers['Content-Type']
  delete headers['content-type']
}

export function formatKnowledgeUploadError(error: unknown, fallback: string): string {
  if (!error || typeof error !== 'object') return fallback

  const payload = error as Record<string, unknown>
  const nestedError = payload.error && typeof payload.error === 'object'
    ? payload.error as Record<string, unknown>
    : null
  const message = typeof nestedError?.message === 'string'
    ? nestedError.message
    : typeof payload.message === 'string'
      ? payload.message
      : fallback
  const details = typeof nestedError?.details === 'string'
    ? nestedError.details
    : typeof payload.details === 'string'
      ? payload.details
      : ''

  return details && details !== message ? `${message}: ${details}` : message
}

export function getUploadFileDiagnostics(file: File | null | undefined) {
  return {
    name: file?.name || '',
    size: typeof file?.size === 'number' ? file.size : 0,
    type: file?.type || '',
    webkitRelativePath: (file as (File & { webkitRelativePath?: string }) | null | undefined)?.webkitRelativePath || '',
  }
}

export async function snapshotUploadFiles(files: File[]): Promise<File[]> {
  return Promise.all(files.map(async (file) => {
    const snapshot = new File([await file.arrayBuffer()], file.name, {
      type: file.type,
      lastModified: file.lastModified,
    })
    const relativePath = (file as File & { webkitRelativePath?: string }).webkitRelativePath
    if (relativePath) {
      Object.defineProperty(snapshot, 'webkitRelativePath', { value: relativePath })
    }
    return snapshot
  }))
}

export function isKnowledgeFileUploadComplete(response: unknown): boolean {
  if (!response || typeof response !== 'object') return false

  const payload = response as KnowledgeUploadResponse
  return payload.success === true
    && !payload.error
}
