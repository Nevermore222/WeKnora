/**
 * Copy text in both secure and insecure browser contexts.
 *
 * navigator.clipboard is unavailable or rejected by browsers on HTTP pages.
 * Keep the legacy textarea path as a compatibility fallback for those pages.
 */
export async function copyText(text: string): Promise<void> {
  const clipboard = typeof navigator !== 'undefined' ? navigator.clipboard : undefined

  if (clipboard?.writeText) {
    try {
      await clipboard.writeText(text)
      return
    } catch {
      // Fall through to the legacy path for insecure contexts and WebViews.
    }
  }

  if (typeof document === 'undefined' || !document.body) {
    throw new Error('Clipboard is unavailable')
  }

  const textArea = document.createElement('textarea')
  textArea.value = text
  textArea.setAttribute('readonly', '')
  textArea.style.position = 'fixed'
  textArea.style.top = '-9999px'
  textArea.style.opacity = '0'
  textArea.style.pointerEvents = 'none'
  document.body.appendChild(textArea)

  try {
    textArea.select()
    if (!document.execCommand('copy')) {
      throw new Error('Copy command was rejected')
    }
  } finally {
    document.body.removeChild(textArea)
  }
}
