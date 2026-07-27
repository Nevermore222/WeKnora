const HIDDEN_LINK_PATTERNS: RegExp[] = [
  /github\.com/i,
  /tencent\.com/i,
  /weixin\.qq\.com/i,
  /weknora/i,
]

/** Hide upstream GitHub / Tencent / WeKnora outbound links in product UI. */
export function shouldShowExternalLink(url: string | null | undefined): boolean {
  if (!url) return false
  const normalized = url.trim()
  if (!normalized) return false
  return !HIDDEN_LINK_PATTERNS.some((pattern) => pattern.test(normalized))
}

export function filterExternalLinks<T extends { url: string }>(links: readonly T[]): T[] {
  return links.filter((link) => shouldShowExternalLink(link.url))
}
