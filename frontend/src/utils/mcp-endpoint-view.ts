/** Build the URL that external MCP clients should use for a published endpoint. */
export function buildMcpEndpointUrl(origin: string, path: string): string {
  const base = origin.trim().replace(/\/+$/, '')
  const normalizedPath = `/${path.trim().replace(/^\/+/, '')}`
  return `${base}${normalizedPath}`
}

/**
 * Resolve the externally reachable origin shown on the MCP publishing page.
 * A runtime value is preferred so the same image can be used in different
 * deployments; the build-time value is useful for static hosting, and the
 * browser origin remains the safe local-development fallback.
 */
export function resolveMcpPublicOrigin(browserOrigin: string, configuredOrigin?: string): string {
  const configured = configuredOrigin?.trim().replace(/\/+$/, '')
  return configured || browserOrigin.trim().replace(/\/+$/, '')
}
