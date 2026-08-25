export type BrandProfileId = 'public' | 'internal'

const viteEnv = (import.meta as ImportMeta & {
  env?: { VITE_BRAND_PROFILE?: string }
}).env

export const brandProfileId: BrandProfileId = viteEnv?.VITE_BRAND_PROFILE === 'internal'
  ? 'internal'
  : 'public'

export const brandDisplayName = brandProfileId === 'internal' ? 'Knavis' : 'Xelora'

const genericPublicBrandPattern =
  /(^|[^A-Za-z0-9_/-])Xelora(?:-(?:app|ui))?(?![A-Za-z0-9_-]|\s+Cloud\b)/gi

export function localizeBrandText(value: string): string {
  if (brandProfileId !== 'internal') return value
  return value.replace(genericPublicBrandPattern, (_match, prefix: string) => `${prefix}${brandDisplayName}`)
}
