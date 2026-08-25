<template>
  <div class="login-layout login-layout--app" :class="{ 'login-layout--register': isRegisterMode }">
    <LoginParticleBackground />

    <!-- Logo - Top Left -->
    <div class="header-logo" aria-hidden="true">
      <img :src="headerLogoSrc" :alt="brandProfile.displayName" class="logo-image" />
    </div>

    <!-- Header Links - Top Right -->
    <div class="header-links">
      <div class="language-switch">
        <button @click="toggleLanguageMenu" class="header-link" :title="currentLangOption?.label">
          <span class="lang-flag-icon">{{ currentLangOption?.flag }}</span>
          <span class="link-text">{{ currentLangOption?.shortLabel }}</span>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"
            stroke-linecap="round">
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </button>

        <!-- Language Dropdown -->
        <div v-if="showLanguageMenu" class="language-dropdown">
          <div v-for="lang in languageOptions" :key="lang.value" @click="selectLanguage(lang.value)"
            class="language-option" :class="{ active: currentLanguage === lang.value }">
            <span class="lang-flag">{{ lang.flag }}</span>
            <span class="lang-label">{{ lang.label }}</span>
            <span v-if="currentLanguage === lang.value" class="check-icon">✓</span>
          </div>
        </div>
      </div>
    </div>

    <main class="login-shell">
      <div class="login-panel">
        <div class="login-panel__glow" aria-hidden="true" />
        <div class="login-panel__grain" aria-hidden="true" />

        <div
          v-if="isDefaultEnterpriseRequired() && defaultEnterpriseConnectionState !== 'ready'"
          class="enterprise-connection-status"
          :class="`enterprise-connection-status--${defaultEnterpriseConnectionState}`"
          aria-live="polite"
        >
          <t-icon :name="defaultEnterpriseConnectionIcon" class="enterprise-connection-status__icon" />
          <div class="enterprise-connection-status__body">
            <strong>{{ defaultEnterpriseConnectionStatusText }}</strong>
            <span v-if="defaultEnterpriseConnectionMessage">{{ defaultEnterpriseConnectionMessage }}</span>
            <span v-else-if="defaultEnterpriseServerURL">{{ defaultEnterpriseServerURL }}</span>
          </div>
          <t-button
            v-if="defaultEnterpriseConnectionState === 'error'"
            theme="default"
            variant="text"
            size="small"
            @click="retryEnterpriseConnection"
          >
            {{ $t('enterprise.retryConnection') }}
          </t-button>
        </div>

        <!-- Login Card -->
        <div class="form-card" v-if="!isRegisterMode">
          <div class="form-header">
            <div class="form-brand">
              <span class="form-brand__mark">{{ brandProfile.displayName.charAt(0) }}</span>
              <span class="form-brand__name">{{ brandProfile.displayName }}</span>
            </div>
            <h2 class="form-title">{{ $t('auth.login') }}</h2>
            <p class="form-welcome">{{ $t('auth.subtitle') }}</p>
            <div class="feature-pills">
              <span class="feature-pill">{{ $t('platform.rag') }}</span>
              <span class="feature-pill">{{ $t('platform.agent') }}</span>
              <span class="feature-pill">{{ $t('platform.wiki') }}</span>
            </div>
          </div>

          <div class="form-content">
            <t-form ref="formRef" :data="formData" :rules="formRules" @submit="handleLogin" layout="vertical">
              <t-form-item :label="$t('auth.email')" name="email">
                <t-input v-model="formData.email" :placeholder="$t('auth.emailPlaceholder')" type="text"
                  autocomplete="email" size="large" :disabled="loading" class="login-field">
                  <template #prefix-icon>
                    <t-icon name="mail" class="login-field__icon" />
                  </template>
                </t-input>
              </t-form-item>

              <t-form-item :label="$t('auth.password')" name="password">
                <t-input v-model="formData.password" :placeholder="$t('auth.passwordPlaceholder')" type="password"
                  size="large" :disabled="loading" class="login-field" @enter="handleLogin">
                  <template #prefix-icon>
                    <t-icon name="lock-on" class="login-field__icon" />
                  </template>
                </t-input>
              </t-form-item>

              <t-button
                type="submit"
                theme="primary"
                size="large"
                block
                :loading="loading"
                :disabled="loading || (isDefaultEnterpriseRequired() && !defaultEnterpriseConnectionReady)"
                class="submit-button"
              >
                {{ loading ? $t('auth.loggingIn') : $t('auth.login') }}
              </t-button>

              <div class="register-cta" v-if="registrationEnabled">
                <p class="register-cta__text">
                  {{ $t('auth.firstTime', { brand: brandProfile.displayName }) }}
                  <button type="button" class="register-cta__link" :disabled="loading" @click="toggleMode">
                    {{ $t('auth.createAccount') }}
                  </button>
                </p>
              </div>

              <div v-if="oidcEnabled" class="oidc-divider">
                <span>{{ $t('auth.orContinueWith') }}</span>
              </div>

              <t-button
                v-if="oidcEnabled"
                theme="default"
                size="large"
                block
                :loading="oidcLoading"
                :disabled="loading || (isDefaultEnterpriseRequired() && !defaultEnterpriseConnectionReady)"
                class="oidc-button"
                @click="handleOIDCLogin"
              >
                {{ oidcLoading ? $t('auth.redirectingToOIDC') : oidcLoginText }}
              </t-button>
            </t-form>
          </div>
        </div>

        <!-- Register Card -->
        <div class="form-card" v-if="isRegisterMode && (registrationEnabled || inviteLookup)">
          <div v-if="inviteLookup" class="invite-banner">
            <t-icon name="link" class="invite-banner__icon" />
            <div class="invite-banner__text">
              <div class="invite-banner__title">
                {{ $t('inviteRegister.bannerTitle', { tenant: inviteLookup.tenant_name || '' }) }}
              </div>
              <div class="invite-banner__hint">
                {{ $t('inviteRegister.bannerHint') }}
              </div>
            </div>
          </div>
          <div v-else-if="inviteLookupError" class="invite-banner invite-banner--error">
            {{ inviteLookupError }}
          </div>

          <div class="form-header">
            <div class="form-brand">
              <span class="form-brand__mark">{{ brandProfile.displayName.charAt(0) }}</span>
              <span class="form-brand__name">{{ brandProfile.displayName }}</span>
            </div>
            <h2 class="form-title">{{ $t('auth.createAccount') }}</h2>
            <p class="form-subtitle">{{ $t('auth.registerSubtitle') }}</p>
            <div class="feature-pills">
              <span class="feature-pill">{{ $t('platform.independentTenant') }}</span>
              <span class="feature-pill">{{ $t('platform.fullApiAccess') }}</span>
            </div>
          </div>

          <div class="form-content">
            <t-form ref="registerFormRef" :data="registerData" :rules="registerRules" @submit="handleRegister"
              layout="vertical">
              <t-form-item :label="$t('auth.username')" name="username">
                <t-input v-model="registerData.username" :placeholder="$t('auth.usernamePlaceholder')" size="large"
                  :disabled="loading" class="login-field">
                  <template #prefix-icon>
                    <t-icon name="user" class="login-field__icon" />
                  </template>
                </t-input>
              </t-form-item>

              <t-form-item :label="$t('auth.email')" name="email">
                <t-input v-model="registerData.email" :placeholder="$t('auth.emailPlaceholder')" type="text"
                  autocomplete="email" size="large" :disabled="loading" class="login-field">
                  <template #prefix-icon>
                    <t-icon name="mail" class="login-field__icon" />
                  </template>
                </t-input>
              </t-form-item>

              <t-form-item :label="$t('auth.password')" name="password">
                <t-input v-model="registerData.password" :placeholder="$t('auth.passwordPlaceholder')" type="password"
                  size="large" :disabled="loading" class="login-field">
                  <template #prefix-icon>
                    <t-icon name="lock-on" class="login-field__icon" />
                  </template>
                </t-input>
              </t-form-item>

              <t-form-item :label="$t('auth.confirmPassword')" name="confirmPassword">
                <t-input v-model="registerData.confirmPassword" :placeholder="$t('auth.confirmPasswordPlaceholder')"
                  type="password" size="large" :disabled="loading" class="login-field" @enter="handleRegister">
                  <template #prefix-icon>
                    <t-icon name="secured" class="login-field__icon" />
                  </template>
                </t-input>
              </t-form-item>

              <t-button
                type="submit"
                theme="primary"
                size="large"
                block
                :loading="loading"
                :disabled="loading || (isDefaultEnterpriseRequired() && !defaultEnterpriseConnectionReady)"
                class="submit-button"
              >
                {{ loading ? $t('auth.registering') : $t('auth.register') }}
              </t-button>
            </t-form>

            <div class="form-footer">
              <span>{{ $t('auth.haveAccount') }}</span>
              <button type="button" class="link-button" :disabled="loading" @click="toggleMode">
                {{ $t('auth.backToLogin') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, nextTick, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoleLabel } from '@/composables/useRoleLabel'
import { notifyLoginSuccess } from '@/utils/loginNotify'
import LoginParticleBackground from '@/components/auth/LoginParticleBackground.vue'
import {
  login,
  register,
  getOIDCAuthorizationURL,
  getOIDCConfig,
  autoSetup,
  getAuthConfig,
  userInfoFromApi,
  getInvitationByToken,
  registerByInvite,
  type InviteLookup,
} from '@/api/auth'
import {
  getDesktopAuthConfig,
  loginDesktopProfile,
  registerDesktopProfile,
  registerDesktopProfileByInvite,
} from '@/api/desktop-remote'
import { useAuthStore } from '@/stores/auth'
import { getDesktopBootstrap, getRuntimeContext, isDefaultEnterpriseRequired } from '@/utils/api-context'
import {
  classifyDefaultEnterpriseConnectionError,
  getDefaultEnterpriseErrorMessage,
  type DefaultEnterpriseConnectionFailureKind,
} from '@/utils/default-enterprise-bootstrap'
import { ensureDefaultEnterpriseRuntime } from '@/utils/default-enterprise-runtime'
import { useI18n } from 'vue-i18n'

import { brandProfile } from '@/config/brandProfile'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { t, tm, locale } = useI18n()
const { formatRole, roleIcon } = useRoleLabel()
const headerLogoSrc = computed(() => brandProfile.darkLogo)

// Form references
const formRef = ref()
const registerFormRef = ref()

// State management
const loading = ref(false)
const oidcLoading = ref(false)
const isRegisterMode = ref(false)
const showLanguageMenu = ref(false)
const oidcEnabled = ref(false)
const oidcProviderName = ref('')
// registrationEnabled defaults to true so that on first paint the Register
// link is visible; the actual mode is fetched from /auth/config in onMounted.
// In invite_only mode the link/card are hidden.
const registrationEnabled = ref(true)
type DefaultEnterpriseConnectionState = 'connecting' | 'ready' | 'error'
const defaultEnterpriseConnectionState = ref<DefaultEnterpriseConnectionState>(
  isDefaultEnterpriseRequired() ? 'connecting' : 'ready',
)
const defaultEnterpriseConnectionMessage = ref('')
const defaultEnterpriseServerURL = computed(() => getDesktopBootstrap()?.default_enterprise_server_url || '')
const defaultEnterpriseConnectionReady = computed(() =>
  !isDefaultEnterpriseRequired() || defaultEnterpriseConnectionState.value === 'ready',
)
const defaultEnterpriseConnectionIcon = computed(() => {
  if (defaultEnterpriseConnectionState.value === 'ready') return 'check-circle'
  if (defaultEnterpriseConnectionState.value === 'error') return 'error-circle'
  return 'loading'
})
const defaultEnterpriseConnectionStatusText = computed(() =>
  t(`enterprise.connectionStatus.${defaultEnterpriseConnectionState.value}`),
)

// invite-link state. When the URL carries ?token=xxx we resolve it to
// the originating tenant + role and switch the form into a "register
// via invitation" mode. The token bypasses the normal invite_only
// gate — possessing it IS the authorisation. Submitting the register
// form with this set hits /auth/register-by-invite (auto-login on
// success) instead of /auth/register.
const inviteToken = ref('')
const inviteLookup = ref<InviteLookup | null>(null)
const inviteLookupError = ref('')
const inviteLookupLoading = ref(false)

// Language options
const languageOptions = [
  { value: 'zh-CN', label: '简体中文', shortLabel: '中文', flag: '🇨🇳' },
  { value: 'en-US', label: 'English', shortLabel: 'EN', flag: '🇺🇸' },
  { value: 'ru-RU', label: 'Русский', shortLabel: 'RU', flag: '🇷🇺' },
  { value: 'ko-KR', label: '한국어', shortLabel: '한국어', flag: '🇰🇷' },
  { value: 'ja-JP', label: '日本語', shortLabel: '日本語', flag: '🇯🇵' },
  { value: 'vi-VN', label: 'Tiếng Việt', shortLabel: 'VI', flag: '🇻🇳' }
]

const currentLanguage = computed(() => locale.value)
const oidcLoginText = computed(() => {
  if (oidcProviderName.value) {
    return t('auth.oidcLoginWithProvider', { provider: oidcProviderName.value })
  }
  return t('auth.oidcLogin')
})
const currentLangOption = computed(() => languageOptions.find(l => l.value === currentLanguage.value))

// Login form data
const formData = reactive<{ [key: string]: any }>({
  email: '',
  password: '',
})

// Register form data
const registerData = reactive<{ [key: string]: any }>({
  username: '',
  email: '',
  password: '',
  confirmPassword: ''
})

// Login form validation rules
const formRules = computed(() => ({
  email: [
    { required: true, message: t('auth.emailRequired'), type: 'error' },
    { email: true, message: t('auth.emailInvalid'), type: 'error' }
  ],
  password: [
    { required: true, message: t('auth.passwordRequired'), type: 'error' },
    { min: 8, message: t('auth.passwordMinLength'), type: 'error' },
    { max: 32, message: t('auth.passwordMaxLength'), type: 'error' },
    { pattern: /[a-zA-Z]/, message: t('auth.passwordMustContainLetter'), type: 'error' },
    { pattern: /\d/, message: t('auth.passwordMustContainNumber'), type: 'error' }
  ]
}))

// Register form validation rules
const registerRules = computed(() => ({
  username: [
    { required: true, message: t('auth.usernameRequired'), type: 'error' },
    { min: 2, message: t('auth.usernameMinLength'), type: 'error' },
    { max: 20, message: t('auth.usernameMaxLength'), type: 'error' },
    {
      pattern: /^[a-zA-Z0-9_\u4e00-\u9fa5]+$/,
      message: t('auth.usernameInvalid'),
      type: 'error'
    }
  ],
  email: [
    { required: true, message: t('auth.emailRequired'), type: 'error' },
    { email: true, message: t('auth.emailInvalid'), type: 'error' }
  ],
  password: [
    { required: true, message: t('auth.passwordRequired'), type: 'error' },
    { min: 8, message: t('auth.passwordMinLength'), type: 'error' },
    { max: 32, message: t('auth.passwordMaxLength'), type: 'error' },
    { pattern: /[a-zA-Z]/, message: t('auth.passwordMustContainLetter'), type: 'error' },
    { pattern: /\d/, message: t('auth.passwordMustContainNumber'), type: 'error' }
  ],
  confirmPassword: [
    { required: true, message: t('auth.confirmPasswordRequired'), type: 'error' },
    {
      validator: (val: string) => val === registerData.password,
      message: t('auth.passwordMismatch'),
      type: 'error'
    }
  ]
}))

// Toggle login/register mode
const toggleMode = () => {
  isRegisterMode.value = !isRegisterMode.value

  Object.keys(registerData).forEach(key => {
    (registerData as any)[key] = ''
  })

  window.scrollTo(0, 0)
}

// Toggle language menu
const toggleLanguageMenu = () => {
  showLanguageMenu.value = !showLanguageMenu.value
}

// Select language
const selectLanguage = (lang: string) => {
  locale.value = lang
  localStorage.setItem('locale', lang)
  showLanguageMenu.value = false
  MessagePlugin.success(t('language.languageSaved'))
}

// Close language menu when clicking outside
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (!target.closest('.language-switch')) {
    showLanguageMenu.value = false
  }
}

// Add click outside listener
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})

const persistLoginResponse = async (response: any) => {
  // Backend renamed `tenant` to `active_tenant` and added `memberships`
  // when tenant-level RBAC landed (issue #1303). The two are otherwise
  // identical — `active_tenant` is the tenant whose ID is encoded in the
  // JWT, defaulting to the user's home tenant on a fresh login.
  const activeTenant = response.active_tenant || response.tenant
  if (response.user && activeTenant && response.token) {
    // user.tenant_id must be the user's HOME tenant (the immutable row
    // on the users table); useHomeTenant() and the home-badge logic both
    // assume so. The ACTIVE tenant (which can differ from home when the
    // server honoured a remembered last-active-tenant preference) is
    // expressed separately via setSelectedTenant below.
    const homeTenantIdRaw = response.user.tenant_id ?? activeTenant.id
    authStore.setUser(userInfoFromApi(response.user, homeTenantIdRaw))
    authStore.setToken(response.token)
    if (response.refresh_token) {
      authStore.setRefreshToken(response.refresh_token)
    }
    authStore.setTenant({
      id: String(activeTenant.id) || '',
      name: activeTenant.name || '',
      api_key: activeTenant.api_key || '',
      owner_id: response.user.id || '',
      created_at: activeTenant.created_at || new Date().toISOString(),
      updated_at: activeTenant.updated_at || new Date().toISOString()
    })
    if (Array.isArray(response.memberships)) {
      authStore.setMemberships(response.memberships)
    }
    // If the backend dropped us into a non-home tenant (honoured a
    // remembered "last active tenant" preference), set the override so
    // subsequent requests carry X-Tenant-ID and the UI stays consistent.
    // Otherwise clear any stale override left in localStorage by a
    // previous session for a different account.
    const activeIdNum = Number(activeTenant.id)
    const homeIdNum = Number(homeTenantIdRaw)
    if (Number.isFinite(activeIdNum) && Number.isFinite(homeIdNum) && activeIdNum !== homeIdNum) {
      authStore.setSelectedTenant(activeIdNum, activeTenant.name || null)
    } else {
      authStore.setSelectedTenant(null, null)
    }
  }

  await nextTick()
  router.replace('/platform/knowledge-bases')
}

const getBackendOIDCRedirectURI = () => `${window.location.origin}/api/v1/auth/oidc/callback`

const loadOIDCConfig = async () => {
  try {
    const response = await getOIDCConfig()
    oidcEnabled.value = !!response.success && !!response.enabled
    oidcProviderName.value = response.provider_display_name || ''
  } catch {
    oidcEnabled.value = false
    oidcProviderName.value = ''
  }
}

// loadAuthConfig fetches /auth/config and caches whether self-service
// registration is allowed. Failures fall back to "enabled" so a transient
// network glitch doesn't lock new users out of an open deployment.
const loadAuthConfig = async () => {
  try {
    const runtimeContext = getRuntimeContext()
    const response = runtimeContext.kind === 'enterprise'
      ? await getDesktopAuthConfig(runtimeContext.profileId)
      : await getAuthConfig()
    registrationEnabled.value = response.registration_mode !== 'invite_only'
  } catch {
    registrationEnabled.value = true
  }
}

const formatDefaultEnterpriseConnectionError = (error: unknown) => {
  const kind: DefaultEnterpriseConnectionFailureKind = classifyDefaultEnterpriseConnectionError(error)
  const detail = getDefaultEnterpriseErrorMessage(error).trim() || t('enterprise.connectionErrors.unknownDetail')
  const url = defaultEnterpriseServerURL.value || t('enterprise.connectionErrors.unknownURL')
  return t(`enterprise.connectionErrors.${kind}`, { url, message: detail })
}

const markDefaultEnterpriseConnectionReady = () => {
  const shouldNotify = defaultEnterpriseConnectionState.value !== 'ready'
  defaultEnterpriseConnectionState.value = 'ready'
  defaultEnterpriseConnectionMessage.value = ''
  if (shouldNotify) {
    MessagePlugin.success({
      content: t('enterprise.connectionStatus.ready'),
      duration: 3000,
      placement: 'top',
    })
  }
}

const ensureEnterpriseConnection = async (showToast = false): Promise<boolean> => {
  if (!isDefaultEnterpriseRequired()) {
    defaultEnterpriseConnectionState.value = 'ready'
    defaultEnterpriseConnectionMessage.value = ''
    return true
  }

  if (getRuntimeContext().kind === 'enterprise') {
    markDefaultEnterpriseConnectionReady()
    return true
  }

  defaultEnterpriseConnectionState.value = 'connecting'
  defaultEnterpriseConnectionMessage.value = ''

  try {
    const context = await ensureDefaultEnterpriseRuntime({ switchContext: true })
    if (!context || context.kind !== 'enterprise') {
      throw new Error('default enterprise server is not configured')
    }
    markDefaultEnterpriseConnectionReady()
    return true
  } catch (error) {
    defaultEnterpriseConnectionState.value = 'error'
    defaultEnterpriseConnectionMessage.value = formatDefaultEnterpriseConnectionError(error)
    if (showToast) MessagePlugin.error(defaultEnterpriseConnectionMessage.value)
    return false
  }
}

const retryEnterpriseConnection = async () => {
  if (await ensureEnterpriseConnection(true)) {
    await Promise.all([loadOIDCConfig(), loadAuthConfig()])
  }
}

const handleOIDCLogin = async () => {
  try {
    if (!(await ensureEnterpriseConnection(true))) return
    oidcLoading.value = true
    const response = await getOIDCAuthorizationURL(getBackendOIDCRedirectURI())
    const authorizationURL = response.authorization_url

    if (!response.success || !authorizationURL) {
      MessagePlugin.error(response.message || t('auth.oidcLoginFailed'))
      return
    }

    window.location.href = authorizationURL
  } catch (error: any) {
    console.error('OIDC 登录跳转失败:', error)
    MessagePlugin.error(error.message || t('auth.oidcLoginFailed'))
  } finally {
    oidcLoading.value = false
  }
}

// Handle login
const handleLogin = async () => {
  try {
    const valid = await formRef.value?.validate()
    if (valid !== true) return

    loading.value = true

    let runtimeContext = getRuntimeContext()
    if (isDefaultEnterpriseRequired() && runtimeContext.kind !== 'enterprise') {
      if (!(await ensureEnterpriseConnection(true))) return
      runtimeContext = getRuntimeContext()
    }
    if (runtimeContext.kind === 'enterprise') {
      const snapshot = await loginDesktopProfile(runtimeContext.profileId, {
        email: formData.email,
        password: formData.password,
      })
      authStore.applyDesktopIdentitySnapshot(snapshot)
      await nextTick()
      router.replace('/platform/knowledge-bases')
      return
    }

    const response = await login({
      email: formData.email,
      password: formData.password,
    })

    if (response.success) {
      await persistLoginResponse(response)
      notifyLoginSuccess(response, t, tm, formatRole, roleIcon)
    } else {
      MessagePlugin.error(response.message || t('auth.loginError'))
    }
  } catch (error: any) {
    console.error('登录错误:', error)
    MessagePlugin.error(error.message || t('auth.loginErrorRetry'))
  } finally {
    loading.value = false
  }
}

// Handle registration. Dispatches based on whether the user arrived
// with a share-link token: with token -> register-by-invite (auto-
// login on success); without -> the normal self-service register
// (drops back to the login form for the user to sign in).
const handleRegister = async () => {
  try {
    const valid = await registerFormRef.value?.validate()
    if (valid !== true) return

    loading.value = true

    let runtimeContext = getRuntimeContext()
    if (isDefaultEnterpriseRequired() && runtimeContext.kind !== 'enterprise') {
      if (!(await ensureEnterpriseConnection(true))) return
      runtimeContext = getRuntimeContext()
    }

    if (inviteToken.value) {
      const invitePayload = {
        token: inviteToken.value,
        username: registerData.username,
        email: registerData.email,
        password: registerData.password,
      }
      const response = runtimeContext.kind === 'enterprise'
        ? await registerDesktopProfileByInvite(runtimeContext.profileId, invitePayload)
        : await registerByInvite(invitePayload)
      if (!response.success) {
        MessagePlugin.error(response.message || t('auth.registerFailed'))
        return
      }
      MessagePlugin.success(t('auth.registerSuccess'))
      // register-by-invite returns the same shape as login (token +
      // active_tenant + memberships), so reuse the login persistence
      // path — same store writes, same redirect target.
      await persistLoginResponse(response)
      return
    }

    const response = runtimeContext.kind === 'enterprise'
      ? await registerDesktopProfile(runtimeContext.profileId, {
          username: registerData.username,
          email: registerData.email,
          password: registerData.password
        })
      : await register({
          username: registerData.username,
          email: registerData.email,
          password: registerData.password
        })

    if (response.success) {
      MessagePlugin.success(t('auth.registerSuccess'))

      // Switch to login mode and fill in email
      isRegisterMode.value = false
      formData.email = registerData.email

      // Clear register form
      Object.keys(registerData).forEach(key => {
        (registerData as any)[key] = ''
      })
    } else {
      MessagePlugin.error(response.message || t('auth.registerFailed'))
    }
  } catch (error: any) {
    console.error('注册错误:', error)
    MessagePlugin.error(error.message || t('auth.registerError'))
  } finally {
    loading.value = false
  }
}

// Check if already logged in; for lite edition, attempt transparent auto-setup
onMounted(async () => {
  // Share-link landing: ?token=xxx switches the form into invite-
  // register mode before any other auto-flow (logged-in redirect /
  // auto-setup / OIDC) gets a chance to redirect. Resolution failure
  // surfaces inline; the user can still log in normally if they
  // already have an account. We check this BEFORE the isLoggedIn
  // redirect so an existing session doesn't bounce the user to
  // /platform (and possibly back to /login if the session is stale),
  // dropping the invite token along the way.
  const tokenFromQuery = String(route.query.token || '').trim()
  if (tokenFromQuery) {
    inviteToken.value = tokenFromQuery
    inviteLookupLoading.value = true
    try {
      const resp = await getInvitationByToken(tokenFromQuery)
      if (resp.success && resp.data) {
        inviteLookup.value = resp.data
        // Token bypasses invite_only — show the register card even
        // when self-service registration is otherwise disabled.
        registrationEnabled.value = true
        isRegisterMode.value = true
      } else {
        inviteLookupError.value = resp.message || t('inviteRegister.invalidBody')
      }
    } catch {
      inviteLookupError.value = t('inviteRegister.invalidBody')
    } finally {
      inviteLookupLoading.value = false
    }
    // Don't run auto-setup when the user came in via an invite link —
    // they're explicitly trying to register, not bootstrap a Lite
    // single-user instance.
    loadOIDCConfig()
    return
  }

  if (authStore.isLoggedIn) {
    router.replace('/platform/knowledge-bases')
    return
  }

  if (isDefaultEnterpriseRequired() || getRuntimeContext().kind === 'enterprise') {
    if (isDefaultEnterpriseRequired() && !(await ensureEnterpriseConnection())) return
    loadOIDCConfig()
    loadAuthConfig()
    return
  }

  const AUTO_SETUP_FAILED_KEY = 'xelora_auto_setup_failed'
  if (localStorage.getItem(AUTO_SETUP_FAILED_KEY) !== 'true') {
    try {
      const response = await autoSetup()
      if (response.success) {
        authStore.setLiteMode(true)
        await persistLoginResponse(response)
        return
      } else {
        localStorage.setItem(AUTO_SETUP_FAILED_KEY, 'true')
      }
    } catch {
      localStorage.setItem(AUTO_SETUP_FAILED_KEY, 'true')
    }
  }

  loadOIDCConfig()
  loadAuthConfig()
})
</script>

<style lang="less" scoped>
.login-layout {
  position: relative;
  width: 100%;
  min-height: 100%;
  overflow: hidden;
}

.login-layout--app {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100vh;
  height: 100dvh;
  min-height: 100vh;
  min-height: 100dvh;
  padding: 72px 24px 28px;
  box-sizing: border-box;
  background: #050505;
  --login-card-bg: #f3ebe0;
  --login-card-bg-soft: #ebe1d3;
  --login-card-surface: #fffaf3;
  --login-card-border: rgba(120, 90, 55, 0.16);
  --login-card-text: #241c14;
  --login-card-muted: #6d5f4f;
  --login-card-accent: #0a9f56;
}

.login-layout--register {
  padding: 56px 24px 20px;

  .form-card {
    padding: 22px 28px 18px;
  }

  .form-header {
    margin-bottom: 18px;
  }

  .form-brand {
    margin-bottom: 12px;
  }

  .feature-pills {
    margin-top: 12px;
  }

  .form-title {
    font-size: 26px;
    margin-bottom: 6px;
  }

  .form-content :deep(.t-form-item) {
    margin-bottom: 14px;
  }

  .form-content :deep(.t-input) {
    min-height: 44px;
  }

  .form-footer {
    margin-top: 12px;
  }
}

.login-shell {
  position: relative;
  z-index: 2;
  width: 100%;
  max-width: 408px;
}

.login-panel {
  position: relative;
  background:
    radial-gradient(120% 80% at 0% 0%, rgba(255, 220, 170, 0.42) 0%, transparent 55%),
    radial-gradient(100% 70% at 100% 100%, rgba(7, 192, 95, 0.07) 0%, transparent 50%),
    linear-gradient(165deg, #f9f2e8 0%, var(--login-card-bg) 48%, var(--login-card-bg-soft) 100%);
  border-radius: 26px;
  border: 1px solid var(--login-card-border);
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.65) inset,
    0 0 0 1px rgba(255, 255, 255, 0.03),
    0 36px 90px rgba(0, 0, 0, 0.5),
    0 14px 32px rgba(60, 30, 10, 0.16);
  overflow: hidden;
}

.login-panel__glow {
  position: absolute;
  top: -40%;
  left: 20%;
  width: 60%;
  height: 45%;
  background: radial-gradient(circle, rgba(255, 236, 200, 0.7) 0%, transparent 70%);
  pointer-events: none;
}

.login-panel__grain {
  position: absolute;
  inset: 0;
  opacity: 0.035;
  pointer-events: none;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
}

.header-logo {
  position: fixed;
  top: 32px;
  left: 50px;
  z-index: 100;
  pointer-events: none;
  user-select: none;

  .logo-image {
    width: 120px;
    height: auto;
  }
}

.header-links {
  position: fixed;
  top: 28px;
  right: 28px;
  display: flex;
  align-items: center;
  gap: 10px;
  z-index: 100;
}

.header-link {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 9px 15px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.92);
  text-decoration: none;
  font-size: 13px;
  font-weight: 600;
  font-family: var(--app-font-family);
  letter-spacing: 0.2px;
  cursor: pointer;
  position: relative;

  svg {
    flex-shrink: 0;
  }

  .link-text {
    line-height: 1;
  }

  &:hover {
    background: rgba(255, 255, 255, 0.14);
    border-color: rgba(255, 255, 255, 0.22);
    color: #ffffff;
  }
}

.language-switch {
  position: relative;

  button {
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.12);
    color: rgba(255, 255, 255, 0.92);

    .lang-flag-icon {
      font-size: 16px;
      line-height: 1;
      flex-shrink: 0;
    }

    &:hover {
      background: rgba(255, 255, 255, 0.14);
      border-color: rgba(255, 255, 255, 0.22);
    }

    svg:last-child {
      margin-left: 2px;
      flex-shrink: 0;
    }
  }
}

.language-dropdown {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  min-width: 160px;
  background: rgba(255, 255, 255, 0.97);
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  overflow: hidden;
  z-index: 1000;
}

.language-option {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  cursor: pointer;
  font-size: 13px;
  font-family: var(--app-font-family);
  color: var(--td-text-color-primary);

  .lang-flag {
    font-size: 16px;
    flex-shrink: 0;
  }

  .lang-label {
    flex: 1;
  }

  .check-icon {
    color: var(--td-success-color);
    font-weight: 700;
    font-size: 14px;
    flex-shrink: 0;
  }

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
  }

  &.active {
    background: var(--td-success-color-light);
    color: var(--td-brand-color-active);
  }
}

.form-card {
  position: relative;
  z-index: 1;
  background: transparent;
  border-radius: 0;
  padding: 30px 30px 28px;
  box-shadow: none;
  box-sizing: border-box;
  border: none;
  width: 100%;
}

.enterprise-connection-status {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: flex-start;
  gap: 9px;
  margin: 14px 18px 0;
  padding: 10px 12px;
  border: 1px solid rgba(120, 90, 55, 0.14);
  border-radius: 12px;
  background: rgba(255, 250, 243, 0.78);
  color: var(--login-card-muted);
  font-size: 12px;
  line-height: 1.45;

  &--connecting {
    color: #7b6b5b;
  }

  &--ready {
    color: #287447;
    border-color: rgba(40, 116, 71, 0.2);
    background: rgba(235, 250, 240, 0.82);
  }

  &--error {
    color: #9d2b2b;
    border-color: rgba(185, 28, 28, 0.2);
    background: rgba(255, 240, 240, 0.9);
  }
}

.enterprise-connection-status__icon {
  flex-shrink: 0;
  margin-top: 1px;
  font-size: 16px;
}

.enterprise-connection-status__body {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
  overflow-wrap: anywhere;

  strong {
    font-weight: 700;
  }
}

.enterprise-connection-status :deep(.t-button) {
  flex-shrink: 0;
  min-height: 24px;
  padding: 0 4px;
  font-size: 12px;
}

.invite-banner {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  margin-bottom: 18px;
  border-radius: 12px;
  background: rgba(255, 250, 243, 0.85);
  border: 1px solid rgba(120, 90, 55, 0.14);
  color: var(--login-card-text);
}

.invite-banner__icon {
  margin-top: 2px;
  font-size: 18px;
  flex-shrink: 0;
  color: var(--login-card-muted);
}

.invite-banner__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.invite-banner__title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--login-card-text);
}

.invite-banner__hint {
  font-size: 12px;
  color: var(--login-card-muted);
  line-height: 1.5;
}

.invite-banner--error {
  background: rgba(220, 38, 38, 0.06);
  border-color: rgba(220, 38, 38, 0.2);
  color: #b91c1c;
  font-size: 13px;
}

.form-header {
  text-align: center;
  margin-bottom: 26px;
}

.form-brand {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 4px 12px 4px 4px;
  border-radius: 999px;
  background: rgba(255, 252, 246, 0.78);
  border: 1px solid rgba(120, 90, 55, 0.1);
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.7) inset,
    0 2px 8px rgba(90, 60, 20, 0.05);
}

.form-brand__mark {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #ef4444 0%, #f59e0b 55%, #07c05f 100%);
  color: #fff;
  font-size: 13px;
  font-weight: 800;
  letter-spacing: -0.02em;
  box-shadow: 0 3px 10px rgba(239, 68, 68, 0.25);
}

.form-brand__name {
  font-size: 13px;
  font-weight: 700;
  color: var(--login-card-text);
  letter-spacing: 0.04em;
  font-family: var(--app-font-family);
}

.form-title {
  font-size: 30px;
  font-weight: 700;
  color: var(--login-card-text);
  margin: 0 0 8px;
  font-family: var(--app-font-family);
  letter-spacing: -0.03em;
  line-height: 1.15;
}

.form-welcome,
.form-subtitle {
  font-size: 13.5px;
  color: var(--login-card-muted);
  margin: 0 auto;
  max-width: 300px;
  line-height: 1.6;
  font-family: var(--app-font-family);
}

.feature-pills {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 16px;
}

.feature-pill {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(255, 252, 246, 0.72);
  border: 1px solid rgba(120, 90, 55, 0.1);
  color: #6a5a48;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.01em;
  font-family: var(--app-font-family);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.55) inset;
}

.register-cta {
  margin-top: 4px;

  &__text {
    margin: 0;
    text-align: center;
    font-size: 13px;
    color: var(--login-card-muted);
    font-family: var(--app-font-family);
    line-height: 1.5;
  }

  &__link {
    margin-left: 4px;
    padding: 0;
    border: none;
    background: none;
    color: var(--login-card-accent);
    font-size: 13px;
    font-weight: 700;
    font-family: var(--app-font-family);
    cursor: pointer;
    text-decoration: none;

    &:hover:not(:disabled) {
      text-decoration: underline;
    }

    &:disabled {
      opacity: 0.55;
      cursor: not-allowed;
    }
  }
}

.form-content {
  :deep(.t-form-item__label) {
    font-size: 12.5px;
    color: #5a4d3f;
    font-weight: 650;
    margin-bottom: 7px;
    font-family: var(--app-font-family);
    display: block;
    text-align: left;
    letter-spacing: 0.02em;
  }

  :deep(.t-input) {
    border: 1px solid rgba(120, 90, 55, 0.14);
    border-radius: 14px;
    background: linear-gradient(180deg, #fffdf8 0%, #fffaf2 100%);
    min-height: 50px;
    box-shadow:
      0 1px 0 rgba(255, 255, 255, 0.8) inset,
      0 1px 2px rgba(90, 60, 20, 0.04);
    transition: border-color 0.2s, box-shadow 0.2s, background 0.2s;

    &:focus-within {
      border-color: rgba(7, 192, 95, 0.55);
      background: #fffefb;
      box-shadow:
        0 0 0 3px rgba(7, 192, 95, 0.11),
        0 1px 0 rgba(255, 255, 255, 0.85) inset;
    }

    &:hover {
      border-color: rgba(7, 192, 95, 0.38);
    }

    .t-input__prefix {
      margin-right: 2px;
    }

    .t-input__inner {
      border: none !important;
      box-shadow: none !important;
      outline: none !important;
      background: transparent;
      font-size: 14.5px;
      color: var(--login-card-text);
      font-family: var(--app-font-family);

      &::placeholder {
        color: #b0a090;
      }

      &:focus {
        border: none !important;
        box-shadow: none !important;
        outline: none !important;
      }
    }

    .t-input__wrap {
      border: none !important;
      box-shadow: none !important;
    }
  }

  :deep(.login-field__icon) {
    color: #9a8874;
    font-size: 18px;
    transition: color 0.2s;
  }

  :deep(.t-input:focus-within .login-field__icon) {
    color: var(--login-card-accent);
  }

  :deep(.t-form-item) {
    margin-bottom: 15px;

    &:last-child {
      margin-bottom: 0;
    }
  }

  :deep(.t-form-item__control) {
    width: 100%;
  }
}

.submit-button {
  height: 50px;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 700;
  font-family: var(--app-font-family);
  margin: 20px 0 14px;
  letter-spacing: 0.02em;
  box-shadow: 0 10px 28px rgba(7, 192, 95, 0.28);
  transition: transform 0.15s ease, box-shadow 0.15s ease;

  &:hover {
    transform: translateY(-1px);
    box-shadow: 0 14px 32px rgba(7, 192, 95, 0.34);
  }
}

.oidc-divider {
  position: relative;
  margin: 10px 0 8px;
  text-align: center;
  color: #9a8b78;
  font-size: 12px;

  span {
    position: relative;
    z-index: 1;
    padding: 0 12px;
    background: transparent;
    color: #9a8b78;
  }

  &::before {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    top: 50%;
    border-top: 1px solid rgba(120, 90, 55, 0.16);
  }
}

.oidc-button {
  height: 48px;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  background: rgba(255, 250, 243, 0.8) !important;
  border-color: rgba(120, 90, 55, 0.18) !important;
  color: var(--login-card-text) !important;
}

.form-footer {
  text-align: center;
  font-size: 13px;
  color: var(--login-card-muted);
  font-family: var(--app-font-family);
  margin-top: 16px;
  padding-bottom: 0;
  border-bottom: none;

  .link-button {
    margin-left: 4px;
    padding: 0;
    border: none;
    background: none;
    color: var(--login-card-accent);
    font-size: 13px;
    font-weight: 700;
    font-family: var(--app-font-family);
    cursor: pointer;
    text-decoration: none;
    transition: all 0.2s;

    &:hover:not(:disabled) {
      text-decoration: underline;
    }

    &:disabled {
      opacity: 0.55;
      cursor: not-allowed;
    }
  }
}

.login-form-footer {
  border-bottom: none;
  padding-bottom: 8px;
  margin-top: 12px;
}

/* Responsive Design */
@media (max-width: 1024px) {
  .login-layout--app {
    padding: 88px 20px 40px;
  }

  .header-logo {
    top: 26px;
    left: 40px;

    .logo-image {
      width: 100px;
    }
  }

  .header-links {
    top: 22px;
    right: 22px;
    gap: 8px;

    .link-text {
      display: none;
    }

    .header-link {
      padding: 10px;
      gap: 0;
    }
  }
}

@media (max-width: 768px) {
  .login-layout--app {
    padding: 80px 16px 32px;
    justify-content: flex-start;
  }

  .login-shell {
    max-width: 100%;
  }

  .login-panel {
    border-radius: 16px;
  }

  .header-logo {
    top: 22px;
    left: 30px;

    .logo-image {
      width: 80px;
    }
  }

  .header-links {
    top: 18px;
    right: 18px;
    gap: 8px;

    .link-text {
      display: none;
    }

    .header-link {
      padding: 8px 10px;
      font-size: 12px;
    }
  }

  .form-card {
    padding: 24px 20px 28px;
  }

  .form-title {
    font-size: 22px;
  }
}

@media (max-width: 480px) {
  .login-layout--app {
    padding: 72px 12px 24px;
  }

  .header-logo {
    top: 18px;
    left: 20px;

    .logo-image {
      width: 70px;
    }
  }

  .header-links {
    top: 14px;
    right: 14px;
    gap: 6px;
    flex-wrap: wrap;

    .header-link {
      padding: 7px 10px;
      font-size: 11px;
    }
  }

  .form-card {
    padding: 20px 16px 24px;
  }

  .form-header {
    margin-bottom: 24px;
  }
}
</style>

<style lang="less">
html[theme-mode="dark"] {
  .login-layout--app {
    background: #050505;
  }

  .login-layout--app .header-link {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.12);
    color: rgba(255, 255, 255, 0.92);

    &:hover {
      background: rgba(255, 255, 255, 0.14);
      border-color: rgba(255, 255, 255, 0.22);
    }
  }

  .login-layout--app .language-switch button {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.12);
    color: rgba(255, 255, 255, 0.92);

    &:hover {
      background: rgba(255, 255, 255, 0.14);
      border-color: rgba(255, 255, 255, 0.22);
    }
  }

  .login-layout--app .language-dropdown {
    background: rgba(255, 255, 255, 0.97) !important;
    border-color: var(--td-component-stroke) !important;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12) !important;
  }

  .login-layout--app .login-panel {
    background:
      radial-gradient(120% 80% at 0% 0%, rgba(255, 220, 170, 0.45) 0%, transparent 55%),
      radial-gradient(100% 70% at 100% 100%, rgba(7, 192, 95, 0.08) 0%, transparent 50%),
      linear-gradient(165deg, #f8f0e6 0%, #f3ebe0 48%, #ebe1d3 100%) !important;
    border: 1px solid rgba(120, 90, 55, 0.16) !important;
    box-shadow:
      0 1px 0 rgba(255, 255, 255, 0.55) inset,
      0 30px 80px rgba(0, 0, 0, 0.48),
      0 12px 28px rgba(60, 30, 10, 0.18) !important;
  }

  .login-layout--app .form-card {
    background: transparent !important;
    box-shadow: none !important;
  }

  .login-layout--app .form-brand__name,
  .login-layout--app .form-title,
  .login-layout--app .form-content .t-form-item__label,
  .login-layout--app .invite-banner__title {
    color: #241c14 !important;
  }

  .login-layout--app .form-welcome,
  .login-layout--app .form-subtitle,
  .login-layout--app .register-cta__text,
  .login-layout--app .form-footer,
  .login-layout--app .invite-banner__hint {
    color: #6d5f4f !important;
  }

  .login-layout--app .feature-pill {
    background: rgba(255, 250, 243, 0.8) !important;
    border-color: rgba(120, 90, 55, 0.12) !important;
    color: #5d4f40 !important;
  }

  .login-layout--app .form-content .t-input {
    background: #fffaf3 !important;
    border-color: rgba(120, 90, 55, 0.16) !important;

    &:hover {
      border-color: rgba(7, 192, 95, 0.45) !important;
    }

    &:focus-within {
      border-color: rgba(7, 192, 95, 0.65) !important;
      background: #fffdf9 !important;
      box-shadow: 0 0 0 3px rgba(7, 192, 95, 0.12) !important;
    }

    .t-input__inner {
      color: #241c14 !important;
    }
  }

  .login-layout--app .register-cta__link,
  .login-layout--app .form-footer .link-button {
    color: #0a9f56 !important;
  }

  .login-layout--app .oidc-divider {
    color: #9a8b78 !important;

    span {
      background: transparent !important;
      color: #9a8b78 !important;
    }
  }

  .login-layout--app .form-footer {
    border-bottom: none !important;
  }

  .login-layout--app .invite-banner {
    background: rgba(255, 250, 243, 0.85) !important;
    border-color: rgba(120, 90, 55, 0.14) !important;
    color: #241c14 !important;
  }
}
</style>
