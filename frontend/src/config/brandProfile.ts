import { brandDisplayName, brandProfileId } from './brandName'
import type { BrandProfileId } from './brandName'
import xeloraLogo from '@/assets/img/xelora.png'
import xeloraDarkLogo from '@/assets/img/xelora-dark.png'
import knavisLogo from '@/assets/img/knavis.png'
import knavisDarkLogo from '@/assets/img/knavis-dark.png'

export type { BrandProfileId } from './brandName'

type BrandProfile = {
  id: BrandProfileId
  displayName: string
  lightLogo: string
  darkLogo: string
  showXeloraCloudSettings: boolean
}

const isInternalProfile = brandProfileId === 'internal'

// Keep the active brand selection explicit at build time; the package can
// still include assets collected by the existing dynamic image resolver.
export const brandProfile: BrandProfile = isInternalProfile
  ? {
      id: 'internal',
      displayName: brandDisplayName,
      lightLogo: knavisLogo,
      darkLogo: knavisDarkLogo,
      showXeloraCloudSettings: false,
    }
  : {
      id: 'public',
      displayName: brandDisplayName,
      lightLogo: xeloraLogo,
      darkLogo: xeloraDarkLogo,
      showXeloraCloudSettings: true,
    }
