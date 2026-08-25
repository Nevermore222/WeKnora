<template>
  <div class="client-download-page">
    <LoginParticleBackground />

    <header class="download-header">
      <RouterLink to="/login" class="download-brand" aria-label="返回登录页">
        <img :src="brandProfile.darkLogo" :alt="brandProfile.displayName" class="download-brand__logo" />
      </RouterLink>

      <div class="download-header__actions">
        <span class="network-status">
          <span class="network-status__dot" aria-hidden="true" />
          <span>INTERNAL DELIVERY</span>
        </span>
        <RouterLink to="/login" class="login-link">
          <span>登录</span>
          <t-icon name="arrow-right" aria-hidden="true" />
        </RouterLink>
      </div>
    </header>

    <main class="download-shell">
      <section class="download-intro" aria-labelledby="download-title">
        <p class="download-kicker"><span aria-hidden="true" /> DESKTOP CLIENT / 01</p>
        <h1 id="download-title" class="download-title">
          Knavis
          <span>下载中心</span>
        </h1>
        <p class="download-lead">Windows 桌面客户端</p>

        <div class="download-capabilities" aria-label="客户端适用环境">
          <span><t-icon name="desktop" aria-hidden="true" /> Windows 10 / 11</span>
          <span><t-icon name="folder-zip" aria-hidden="true" /> 便携版</span>
          <span><t-icon name="network" aria-hidden="true" /> 局域网部署</span>
        </div>

        <div class="signal-trace" aria-hidden="true">
          <span class="signal-trace__line signal-trace__line--main" />
          <span class="signal-trace__line signal-trace__line--branch" />
          <span class="signal-trace__node signal-trace__node--one" />
          <span class="signal-trace__node signal-trace__node--two" />
          <span class="signal-trace__node signal-trace__node--three" />
          <span class="signal-trace__label">READY TO CONNECT</span>
        </div>
      </section>

      <section class="release-console" aria-labelledby="release-title" aria-live="polite">
        <div class="release-console__topline">
          <div class="release-console__label">
            <span class="release-console__signal" aria-hidden="true" />
            <span>当前发布</span>
          </div>
          <button
            type="button"
            class="refresh-button"
            :disabled="loading"
            title="重新检查服务器上的最新发布包"
            @click="loadRelease()"
          >
            <t-icon name="refresh" :class="{ 'is-spinning': loading }" aria-hidden="true" />
            <span>检查更新</span>
          </button>
        </div>

        <div v-if="loading && !release" class="release-state release-state--loading">
          <t-icon name="loading" aria-hidden="true" />
          <span>正在读取发布信息</span>
        </div>

        <div v-else-if="release" class="release-content">
          <div class="release-heading">
            <div>
              <p class="release-heading__eyebrow">LATEST STABLE BUILD</p>
              <h2 id="release-title">v{{ displayVersion }}</h2>
            </div>
            <span class="release-badge"><t-icon name="check-circle" aria-hidden="true" /> 可用</span>
          </div>

          <div class="release-file">
            <t-icon name="folder-zip" aria-hidden="true" />
            <span :title="release.file">{{ release.file }}</span>
          </div>

          <dl class="release-metadata">
            <div>
              <dt>文件大小</dt>
              <dd>{{ formattedSize }}</dd>
            </div>
            <div>
              <dt>构建编号</dt>
              <dd>{{ release.build || '未标记' }}</dd>
            </div>
            <div>
              <dt>更新时间</dt>
              <dd>{{ formattedDate }}</dd>
            </div>
          </dl>

          <div v-if="release.sha256" class="release-checksum">
            <div>
              <span class="release-checksum__label">SHA-256</span>
              <code :title="release.sha256">{{ shortenedChecksum }}</code>
            </div>
            <button type="button" title="复制 SHA-256" class="icon-button" @click="copyChecksum">
              <t-icon :name="checksumCopied ? 'check' : 'copy'" aria-hidden="true" />
            </button>
          </div>

          <div class="release-actions">
            <a
              class="release-action release-action--primary"
              :href="downloadHref"
              :download="release.file"
            >
              <t-icon name="download" aria-hidden="true" />
              <span>下载客户端</span>
              <t-icon name="arrow-right" aria-hidden="true" />
            </a>
            <button
              type="button"
              class="release-action release-action--secondary"
              :disabled="loading"
              @click="downloadLatest"
            >
              <t-icon name="refresh" :class="{ 'is-spinning': loading }" aria-hidden="true" />
              <span>更新客户端</span>
            </button>
          </div>

          <p class="release-footnote">
            <span class="release-footnote__rule" aria-hidden="true" />
            点击下载将直接获取当前服务器发布包
          </p>
          <div v-if="releaseHistory.length > 1" class="release-history">
            <div class="release-history__header">
              <span>RELEASE CHANNEL HISTORY</span>
              <span>STABLE</span>
            </div>
            <div v-for="item in releaseHistory" :key="`${item.version}-${item.file}`" class="release-history__item">
              <span>v{{ item.version.replace(/^v/i, '') }}</span>
              <span>{{ item.build || item.updatedAt || 'published' }}</span>
            </div>
          </div>
        </div>

        <div v-else class="release-state release-state--error">
          <t-icon name="error-circle" aria-hidden="true" />
          <div>
            <strong>暂时无法读取发布信息</strong>
            <span>{{ errorMessage }}</span>
          </div>
          <button type="button" class="retry-button" @click="loadRelease()">重试</button>
        </div>

        <div v-if="lastChecked" class="release-console__footer">
          <span>SERVER MANIFEST</span>
          <time :datetime="lastChecked">{{ checkedLabel }}</time>
        </div>
      </section>
    </main>

    <footer class="download-footer">
      <span>{{ brandProfile.displayName }} / CLIENT DISTRIBUTION</span>
      <span class="download-footer__mark" aria-hidden="true" />
      <span>INTERNAL NETWORK</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import LoginParticleBackground from '@/components/auth/LoginParticleBackground.vue'
import { brandProfile } from '@/config/brandProfile'

interface ReleaseManifest {
  version: string
  build?: string
  file: string
  sha256?: string
  sizeBytes?: number
  updatedAt?: string
}

interface ParsedReleaseManifest {
  latest: ReleaseManifest
  releases: ReleaseManifest[]
}

const release = ref<ReleaseManifest | null>(null)
const releaseHistory = ref<ReleaseManifest[]>([])
const loading = ref(true)
const errorMessage = ref('请确认服务器已发布客户端清单。')
const lastChecked = ref('')
const checksumCopied = ref(false)
let requestController: AbortController | null = null
let copyResetTimer: number | undefined

const releaseManifestURL = computed(() =>
  brandProfile.id === 'internal' ? '/downloads/channels/stable.json' : '/downloads/latest.json',
)
const displayVersion = computed(() => release.value?.version.replace(/^v/i, '') || '')
const downloadHref = computed(() => {
  if (!release.value) return undefined
  return `/downloads/${encodeURIComponent(release.value.file)}`
})
const formattedSize = computed(() => {
  const bytes = release.value?.sizeBytes
  if (!bytes || bytes <= 0) return '未标记'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
})
const formattedDate = computed(() => {
  const value = release.value?.updatedAt
  if (!value) return '未标记'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
})
const shortenedChecksum = computed(() => {
  const checksum = release.value?.sha256 || ''
  return checksum.length > 28 ? `${checksum.slice(0, 16)}...${checksum.slice(-8)}` : checksum
})
const checkedLabel = computed(() => {
  if (!lastChecked.value) return ''
  return new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit' }).format(
    new Date(lastChecked.value),
  )
})

function isSafeZipName(value: unknown): value is string {
  return typeof value === 'string' && /^[A-Za-z0-9][A-Za-z0-9._-]*\.zip$/i.test(value)
}

function parseManifest(value: unknown): ReleaseManifest {
  if (!value || typeof value !== 'object') throw new Error('发布清单格式无效')
  const candidate = value as Record<string, unknown>
  if (typeof candidate.version !== 'string' || !candidate.version.trim()) {
    throw new Error('发布版本未标记')
  }
  if (!isSafeZipName(candidate.file)) throw new Error('发布文件名无效')

  const sizeValue = typeof candidate.sizeBytes === 'number'
    ? candidate.sizeBytes
    : Number(candidate.sizeBytes || 0)
  return {
    version: candidate.version.trim(),
    build: typeof candidate.build === 'string' ? candidate.build.trim() : '',
    file: candidate.file,
    sha256: typeof candidate.sha256 === 'string' ? candidate.sha256.trim() : '',
    sizeBytes: Number.isFinite(sizeValue) && sizeValue > 0 ? sizeValue : undefined,
    updatedAt: typeof candidate.updatedAt === 'string' ? candidate.updatedAt : '',
  }
}

function parseChannelReleaseEntry(value: unknown, requireIntegrity: boolean): ReleaseManifest {
  if (!value || typeof value !== 'object') throw new Error('Invalid release entry')
  const candidate = value as Record<string, unknown>
  if (
    typeof candidate.version !== 'string'
    || !/^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(candidate.version.trim())
  ) {
    throw new Error('Invalid release version')
  }
  if (!isSafeZipName(candidate.file)) throw new Error('Invalid release filename')

  const sha256 = typeof candidate.sha256 === 'string' ? candidate.sha256.trim() : ''
  if (requireIntegrity && !/^[A-Fa-f0-9]{64}$/.test(sha256)) {
    throw new Error('Invalid release SHA-256')
  }
  const sizeValue = typeof candidate.sizeBytes === 'number'
    ? candidate.sizeBytes
    : Number(candidate.sizeBytes || 0)
  if (requireIntegrity && (!Number.isSafeInteger(sizeValue) || sizeValue <= 0)) {
    throw new Error('Invalid release size')
  }

  return {
    version: candidate.version.trim(),
    build: typeof candidate.build === 'string' ? candidate.build.trim() : '',
    file: candidate.file,
    sha256,
    sizeBytes: Number.isFinite(sizeValue) && sizeValue > 0 ? sizeValue : undefined,
    updatedAt: typeof candidate.updatedAt === 'string'
      ? candidate.updatedAt
      : typeof candidate.publishedAt === 'string' ? candidate.publishedAt : '',
  }
}

function parseChannelManifest(value: unknown): ParsedReleaseManifest {
  if (!value || typeof value !== 'object') throw new Error('Invalid release manifest')
  const candidate = value as Record<string, unknown>
  if (candidate.latest && typeof candidate.latest === 'object') {
    if (
      candidate.schemaVersion !== 1
      || (candidate.channel !== 'stable' && candidate.channel !== 'beta')
    ) {
      throw new Error('Invalid release channel manifest')
    }
    const latest = parseChannelReleaseEntry(candidate.latest, true)
    const releases = Array.isArray(candidate.releases)
      ? candidate.releases.map((item) => parseChannelReleaseEntry(item, true))
      : [latest]
    return { latest, releases }
  }
  const latest = parseManifest(candidate)
  return { latest, releases: [latest] }
}

async function loadRelease() {
  requestController?.abort()
  const controller = new AbortController()
  requestController = controller
  loading.value = true

  try {
    const response = await fetch(`${releaseManifestURL.value}?ts=${Date.now()}`, {
      cache: 'no-store',
      headers: { Accept: 'application/json' },
      signal: controller.signal,
    })
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const parsed = parseChannelManifest(await response.json())
    release.value = parsed.latest
    releaseHistory.value = parsed.releases
    errorMessage.value = ''
    lastChecked.value = new Date().toISOString()
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') return
    if (!release.value) {
      errorMessage.value = error instanceof Error ? error.message : '网络连接失败'
    }
  } finally {
    if (requestController === controller) {
      requestController = null
      loading.value = false
    }
  }
}

async function downloadLatest() {
  await loadRelease()
  if (downloadHref.value) window.location.assign(downloadHref.value)
}

async function copyChecksum() {
  const checksum = release.value?.sha256
  if (!checksum || !navigator.clipboard) return
  try {
    await navigator.clipboard.writeText(checksum)
    checksumCopied.value = true
    if (copyResetTimer) window.clearTimeout(copyResetTimer)
    copyResetTimer = window.setTimeout(() => {
      checksumCopied.value = false
    }, 1800)
  } catch {
    checksumCopied.value = false
  }
}

let previousDocumentTitle = ''
onMounted(() => {
  previousDocumentTitle = document.title
  document.title = `${brandProfile.displayName} 客户端下载`
  void loadRelease()
})

onBeforeUnmount(() => {
  requestController?.abort()
  if (copyResetTimer) window.clearTimeout(copyResetTimer)
  if (previousDocumentTitle) document.title = previousDocumentTitle
})
</script>

<style scoped lang="less">
.client-download-page {
  --download-ink: #f3f0e8;
  --download-muted: rgba(239, 242, 239, 0.78);
  --download-faint: rgba(229, 235, 231, 0.58);
  --download-line: rgba(232, 235, 232, 0.27);
  --download-panel: rgba(14, 18, 18, 0.94);
  --download-teal: #1bb6a4;
  --download-gold: #f2b84b;
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  overflow: hidden;
  color: var(--download-ink);
  background: #050707;
  font-family: var(--app-font-family, "Segoe UI", "Microsoft YaHei", sans-serif);
}

.download-header,
.download-shell,
.download-footer {
  position: relative;
  z-index: 1;
}

.download-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 30px 50px 0;
}

.download-brand {
  display: inline-flex;
  align-items: center;
  min-height: 40px;
}

.download-brand__logo {
  width: 120px;
  height: auto;
}

.download-header__actions,
.network-status,
.login-link {
  display: inline-flex;
  align-items: center;
}

.download-header__actions {
  gap: 28px;
}

.network-status {
  gap: 8px;
  color: var(--download-faint);
  font-size: 10px;
  letter-spacing: 1.4px;
}

.network-status__dot,
.release-console__signal {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--download-teal);
  box-shadow: 0 0 0 4px rgba(27, 182, 164, 0.12), 0 0 14px rgba(27, 182, 164, 0.8);
}

.login-link {
  gap: 7px;
  color: var(--download-muted);
  font-size: 13px;
  text-decoration: none;
  transition: color 180ms ease, transform 180ms ease;
}

.login-link:hover {
  color: var(--download-ink);
  transform: translateX(2px);
}

.download-shell {
  display: grid;
  grid-template-columns: minmax(300px, 0.9fr) minmax(460px, 1.1fr);
  align-items: center;
  gap: 9vw;
  width: min(1120px, calc(100% - 120px));
  min-width: 0;
  min-height: calc(100vh - 126px);
  min-height: calc(100dvh - 126px);
  margin: 0 auto;
  padding: 38px 0 68px;
  box-sizing: border-box;
}

.download-intro {
  animation: intro-in 700ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.download-kicker {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 28px;
  color: var(--download-teal);
  font-size: 11px;
  font-weight: 650;
  letter-spacing: 2px;
}

.download-kicker span {
  width: 30px;
  height: 1px;
  background: var(--download-teal);
  box-shadow: 12px 0 0 rgba(242, 184, 75, 0.55);
}

.download-title {
  max-width: 470px;
  margin: 0;
  color: var(--download-ink);
  font-size: 82px;
  font-weight: 760;
  letter-spacing: 0;
  line-height: 0.95;
}

.download-title span {
  display: block;
  margin-top: 20px;
  color: transparent;
  font-size: 28px;
  font-weight: 500;
  letter-spacing: 0;
  line-height: 1.2;
  -webkit-text-stroke: 1px rgba(243, 240, 232, 0.62);
}

.download-lead {
  margin: 34px 0 0;
  color: var(--download-muted);
  font-size: 17px;
}

.download-capabilities {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 20px;
  margin-top: 24px;
  color: var(--download-muted);
  font-size: 12px;
}

.download-capabilities span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.download-capabilities :deep(.t-icon) {
  color: var(--download-gold);
}

.signal-trace {
  position: relative;
  width: min(370px, 100%);
  height: 88px;
  margin-top: 48px;
  opacity: 0.9;
}

.signal-trace__line {
  position: absolute;
  display: block;
  height: 1px;
  transform-origin: left center;
}

.signal-trace__line--main {
  top: 42px;
  left: 0;
  width: 84%;
  background: linear-gradient(90deg, rgba(27, 182, 164, 0), var(--download-teal) 62%, var(--download-gold));
}

.signal-trace__line--branch {
  top: 42px;
  left: 71%;
  width: 22%;
  background: var(--download-gold);
  transform: rotate(-34deg);
}

.signal-trace__node {
  position: absolute;
  width: 6px;
  height: 6px;
  border: 1px solid #050707;
  border-radius: 50%;
  background: var(--download-gold);
  box-shadow: 0 0 0 3px rgba(242, 184, 75, 0.12);
}

.signal-trace__node--one { top: 39px; left: 55%; }
.signal-trace__node--two { top: 39px; left: 71%; }
.signal-trace__node--three { top: 15px; left: 91%; }

.signal-trace__label {
  position: absolute;
  top: 56px;
  left: 71%;
  color: var(--download-faint);
  font-size: 9px;
  letter-spacing: 1.6px;
  white-space: nowrap;
}

.release-console {
  align-self: center;
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
  padding: 28px 30px 20px;
  border: 1px solid rgba(232, 235, 232, 0.2);
  border-radius: 8px;
  background: linear-gradient(145deg, rgba(255, 255, 255, 0.06), transparent 38%), var(--download-panel);
  box-shadow: 0 28px 72px rgba(0, 0, 0, 0.46), inset 0 1px 0 rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(18px);
  animation: console-in 760ms 100ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.release-console__topline,
.release-console__label,
.refresh-button,
.release-badge,
.release-file,
.release-actions,
.release-footnote,
.release-console__footer {
  display: flex;
  align-items: center;
}

.release-console__topline {
  justify-content: space-between;
  min-width: 0;
  padding-bottom: 22px;
  border-bottom: 1px solid var(--download-line);
}

.release-console__label {
  gap: 11px;
  min-width: 0;
  color: var(--download-ink);
  font-size: 13px;
  font-weight: 650;
}

.refresh-button,
.retry-button,
.icon-button {
  border: 0;
  cursor: pointer;
  font-family: inherit;
}

.refresh-button {
  gap: 7px;
  padding: 7px 9px;
  color: var(--download-muted);
  background: transparent;
  font-size: 12px;
  transition: color 180ms ease, background 180ms ease;
}

.refresh-button:hover:not(:disabled) {
  color: var(--download-ink);
  background: rgba(255, 255, 255, 0.08);
}

button:focus-visible,
a:focus-visible {
  outline: 2px solid var(--download-gold);
  outline-offset: 3px;
}

button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.is-spinning {
  animation: spin 900ms linear infinite;
}

.release-content {
  padding-top: 26px;
}

.release-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.release-heading__eyebrow {
  margin: 0 0 9px;
  color: var(--download-faint);
  font-size: 10px;
  letter-spacing: 1.5px;
}

.release-heading h2 {
  margin: 0;
  color: var(--download-ink);
  font-size: 42px;
  font-weight: 650;
  letter-spacing: 0;
  line-height: 1;
}

.release-badge {
  gap: 6px;
  padding: 7px 10px;
  border: 1px solid rgba(27, 182, 164, 0.25);
  border-radius: 999px;
  color: #8de1d4;
  background: rgba(27, 182, 164, 0.1);
  font-size: 11px;
  white-space: nowrap;
}

.release-file {
  gap: 10px;
  min-width: 0;
  margin-top: 27px;
  padding: 14px 15px;
  border: 1px solid var(--download-line);
  color: var(--download-muted);
  background: rgba(0, 0, 0, 0.16);
  font-family: var(--app-font-family-mono, monospace);
  font-size: 11px;
}

.release-file :deep(.t-icon) {
  flex: 0 0 auto;
  color: var(--download-gold);
  font-size: 18px;
}

.release-file span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.release-metadata {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
  margin: 24px 0 0;
  padding-bottom: 22px;
  border-bottom: 1px solid var(--download-line);
}

.release-metadata div {
  min-width: 0;
}

.release-metadata dt,
.release-metadata dd {
  margin: 0;
}

.release-metadata dt {
  color: var(--download-faint);
  font-size: 10px;
}

.release-metadata dd {
  overflow: hidden;
  margin-top: 8px;
  color: var(--download-ink);
  font-family: var(--app-font-family-mono, monospace);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.release-checksum {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 0 2px;
}

.release-checksum > div {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.release-checksum__label {
  color: var(--download-faint);
  font-size: 10px;
  letter-spacing: 1px;
}

.release-checksum code {
  overflow: hidden;
  color: var(--download-muted);
  font-family: var(--app-font-family-mono, monospace);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.icon-button {
  flex: 0 0 auto;
  width: 28px;
  height: 28px;
  color: var(--download-muted);
  background: transparent;
  transition: color 180ms ease, background 180ms ease;
}

.icon-button:hover {
  color: var(--download-ink);
  background: rgba(255, 255, 255, 0.08);
}

.release-actions {
  gap: 10px;
  margin-top: 25px;
}

.release-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
  min-height: 46px;
  padding: 0 16px;
  border: 1px solid transparent;
  border-radius: 5px;
  box-sizing: border-box;
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 650;
  text-decoration: none;
  transition: transform 180ms ease, border-color 180ms ease, background 180ms ease, box-shadow 180ms ease;
}

.release-action--primary {
  flex: 1 1 auto;
  color: #071311;
  background: var(--download-teal);
  box-shadow: 0 12px 26px rgba(27, 182, 164, 0.18);
}

.release-action--primary:hover {
  background: #36c7b5;
  box-shadow: 0 15px 30px rgba(27, 182, 164, 0.27);
  transform: translateY(-2px);
}

.release-action--secondary {
  flex: 0 0 auto;
  color: var(--download-ink);
  border-color: var(--download-line);
  background: rgba(255, 255, 255, 0.06);
}

.release-action--secondary:hover:not(:disabled) {
  border-color: rgba(242, 184, 75, 0.55);
  background: rgba(242, 184, 75, 0.1);
  transform: translateY(-2px);
}

.release-footnote {
  gap: 10px;
  margin: 18px 0 0;
  color: var(--download-faint);
  font-size: 11px;
}

.release-footnote__rule {
  width: 20px;
  height: 1px;
  background: var(--download-gold);
}

.release-history {
  display: grid;
  gap: 8px;
  margin-top: 22px;
  padding-top: 16px;
  border-top: 1px solid var(--download-line);
  color: var(--download-faint);
  font-family: var(--app-font-family-mono, monospace);
  font-size: 10px;
}

.release-history__header,
.release-history__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.release-history__header {
  color: var(--download-gold);
  font-size: 9px;
  letter-spacing: 1.1px;
}

.release-history__item span:first-child {
  color: var(--download-ink);
}

.release-console__footer {
  justify-content: space-between;
  gap: 12px;
  margin-top: 25px;
  padding-top: 13px;
  border-top: 1px solid var(--download-line);
  color: var(--download-faint);
  font-family: var(--app-font-family-mono, monospace);
  font-size: 9px;
  letter-spacing: 1.2px;
}

.release-state {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 250px;
  color: var(--download-muted);
  font-size: 13px;
}

.release-state--loading :deep(.t-icon) {
  color: var(--download-teal);
}

.release-state--error {
  align-items: flex-start;
  padding: 36px 0;
  color: #f1aaa4;
}

.release-state--error > :deep(.t-icon) {
  flex: 0 0 auto;
  margin-top: 2px;
}

.release-state--error div {
  display: grid;
  gap: 8px;
  min-width: 0;
}

.release-state--error strong {
  color: var(--download-ink);
  font-size: 14px;
}

.release-state--error span {
  color: var(--download-muted);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.retry-button {
  flex: 0 0 auto;
  margin-left: auto;
  padding: 7px 11px;
  border: 1px solid rgba(241, 170, 164, 0.35);
  border-radius: 4px;
  color: #f1aaa4;
  background: transparent;
}

.retry-button:hover {
  background: rgba(241, 170, 164, 0.1);
}

.download-footer {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 50px 24px;
  color: rgba(232, 235, 232, 0.28);
  font-family: var(--app-font-family-mono, monospace);
  font-size: 9px;
  letter-spacing: 1.4px;
}

.download-footer__mark {
  width: 34px;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--download-teal));
}

@keyframes intro-in {
  from { opacity: 0; transform: translateY(18px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes console-in {
  from { opacity: 0; transform: translateY(24px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 820px) {
  .download-header {
    padding: 22px 24px 0;
  }

  .download-header__actions {
    gap: 14px;
  }

  .network-status {
    display: none;
  }

  .download-shell {
    grid-template-columns: minmax(0, 1fr);
    gap: 38px;
    width: min(560px, calc(100% - 48px));
    min-height: 0;
    padding: 40px 0 48px;
  }

  .download-title {
    font-size: 58px;
  }

  .download-title span {
    font-size: 24px;
  }

  .signal-trace {
    margin-top: 30px;
  }

  .release-console {
    order: -1;
    padding: 22px 20px 18px;
    animation: none;
  }

  .download-intro {
    animation: none;
  }

  .download-footer {
    padding: 0 24px 20px;
  }
}

@media (max-width: 500px) {
  .download-brand__logo {
    width: 102px;
  }

  .download-shell {
    width: calc(100% - 32px);
    padding-top: 52px;
  }

  .download-lead {
    margin-top: 25px;
  }

  .download-capabilities {
    display: grid;
    gap: 9px;
  }

  .release-heading h2 {
    font-size: 36px;
  }

  .release-metadata {
    grid-template-columns: 1fr 1fr;
    row-gap: 18px;
  }

  .release-metadata div:last-child {
    grid-column: 1 / -1;
  }

  .release-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .release-action {
    width: 100%;
  }

  .download-footer {
    flex-wrap: wrap;
    line-height: 1.5;
  }
}

@media (prefers-reduced-motion: reduce) {
  .download-intro,
  .release-console {
    animation: none;
  }

  .is-spinning {
    animation: none;
  }
}
</style>
