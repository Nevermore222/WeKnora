# Japanese and Vietnamese UI Internationalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add complete Japanese (`ja-JP`) and Vietnamese (`vi-VN`) interface support to the desktop client, server-hosted web UI, and embedded chat.

**Architecture:** Follow the existing `vue-i18n` layout: standalone main-app locale modules, explicit registration in the main i18n instance, and an isolated embed i18n instance. Treat `en-US` as the structural source of truth and enforce recursive key and interpolation-token parity so neither new locale can silently ship missing messages.

**Tech Stack:** Vue 3, TypeScript, vue-i18n 11, Node test runner, Vite, vue-tsc

---

## File Structure

- Create `frontend/src/i18n/locales/ja-JP.ts`: complete Japanese main-app message bundle.
- Create `frontend/src/i18n/locales/vi-VN.ts`: complete Vietnamese main-app message bundle.
- Create `frontend/i18n-new-locales.test.mjs`: structural, token, and registration regression tests.
- Modify `frontend/src/i18n/index.ts`: register both main-app locales and normalize persisted locale values.
- Modify `frontend/src/views/settings/GeneralSettings.vue`: expose both locales in settings.
- Modify `frontend/src/views/auth/Login.vue`: expose both locales before authentication.
- Modify `frontend/src/components/AgentEmbedChannelPanel.vue`: expose both locales as embed-channel defaults.
- Modify `frontend/src/i18n/embed.ts`: add Japanese/Vietnamese visitor messages and locale normalization.

### Task 1: Lock the locale contract with failing tests

**Files:**
- Create: `frontend/i18n-new-locales.test.mjs`
- Test: `frontend/i18n-new-locales.test.mjs`

- [ ] **Step 1: Write the failing locale contract test**

Create a Node test that imports the three locale modules, recursively compares their keys, compares interpolation tokens, and checks every registration surface:

```js
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import enUS from './src/i18n/locales/en-US.ts'
import jaJP from './src/i18n/locales/ja-JP.ts'
import viVN from './src/i18n/locales/vi-VN.ts'

const leafEntries = (value, prefix = '', entries = new Map()) => {
  for (const [key, child] of Object.entries(value)) {
    const path = prefix ? `${prefix}.${key}` : key
    if (child && typeof child === 'object' && !Array.isArray(child)) {
      leafEntries(child, path, entries)
    } else {
      entries.set(path, child)
    }
  }
  return entries
}

const tokens = (value) =>
  [...String(value).matchAll(/\{[A-Za-z0-9_]+\}/g)].map(([token]) => token).sort()

for (const [tag, locale] of [['ja-JP', jaJP], ['vi-VN', viVN]]) {
  test(`${tag} matches the English key and interpolation contract`, () => {
    const baseline = leafEntries(enUS)
    const candidate = leafEntries(locale)
    assert.deepEqual([...candidate.keys()].sort(), [...baseline.keys()].sort())
    for (const [key, source] of baseline) {
      assert.deepEqual(tokens(candidate.get(key)), tokens(source), `${tag}: ${key}`)
    }
  })
}

test('all UI locale selectors and registries expose Japanese and Vietnamese', () => {
  const files = [
    './src/i18n/index.ts',
    './src/i18n/embed.ts',
    './src/views/settings/GeneralSettings.vue',
    './src/views/auth/Login.vue',
    './src/components/AgentEmbedChannelPanel.vue',
  ].map((path) => readFileSync(new URL(path, import.meta.url), 'utf8'))
  for (const source of files) {
    assert.match(source, /ja-JP/, 'surface must include ja-JP')
    assert.match(source, /vi-VN/, 'surface must include vi-VN')
  }
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd frontend && node --test i18n-new-locales.test.mjs`

Expected: FAIL because `ja-JP.ts` and `vi-VN.ts` do not exist.

- [ ] **Step 3: Commit the failing test**

```bash
git add frontend/i18n-new-locales.test.mjs
git commit -m "test: define Japanese and Vietnamese locale contract"
```

### Task 2: Add complete Japanese and Vietnamese main-app bundles

**Files:**
- Create: `frontend/src/i18n/locales/ja-JP.ts`
- Create: `frontend/src/i18n/locales/vi-VN.ts`
- Test: `frontend/i18n-new-locales.test.mjs`

- [ ] **Step 1: Translate the English baseline into Japanese**

Create `ja-JP.ts` with the exact recursive object shape and leaf count of `en-US.ts`. Translate every user-facing prose leaf into natural Japanese while preserving product names, commands, URLs, HTML tags, and interpolation tokens byte-for-byte. The module form must be:

```ts
export default {
  capabilityDenied: {
    title: '権限がありません',
    message: 'この操作を行う権限がありません。管理者にお問い合わせください。',
    ok: 'OK',
  },
} as const
```

The snippet defines the required module syntax and translation standard. The completed file must contain every recursive leaf reported by `leafEntries(enUS)`; Step 3 is the executable acceptance gate and must report identical sorted paths and tokens before the bundle is committed.

- [ ] **Step 2: Translate the English baseline into Vietnamese**

Create `vi-VN.ts` with the exact recursive object shape and leaf count of `en-US.ts`. Translate every user-facing prose leaf into natural Vietnamese under the same preservation rules. The module form must be:

```ts
export default {
  capabilityDenied: {
    title: 'Không có quyền truy cập',
    message: 'Tài khoản của bạn không có quyền thực hiện thao tác này. Vui lòng liên hệ quản trị viên.',
    ok: 'OK',
  },
} as const
```

The completed file is governed by the same executable `leafEntries(enUS)` acceptance gate; no English-baseline path may be omitted or added and every interpolation token must match.

- [ ] **Step 3: Run the structural test**

Run: `cd frontend && node --test i18n-new-locales.test.mjs`

Expected: key/token tests PASS; registration-surface test still FAILS because the new tags are not registered everywhere.

- [ ] **Step 4: Commit the language bundles**

```bash
git add frontend/src/i18n/locales/ja-JP.ts frontend/src/i18n/locales/vi-VN.ts
git commit -m "feat: add Japanese and Vietnamese translations"
```

### Task 3: Register both locales across the main application

**Files:**
- Modify: `frontend/src/i18n/index.ts`
- Modify: `frontend/src/views/settings/GeneralSettings.vue`
- Modify: `frontend/src/views/auth/Login.vue`
- Modify: `frontend/src/components/AgentEmbedChannelPanel.vue`
- Modify: `frontend/src/i18n/locales/en-US.ts`
- Modify: `frontend/src/i18n/locales/zh-CN.ts`
- Modify: `frontend/src/i18n/locales/ko-KR.ts`
- Modify: `frontend/src/i18n/locales/ru-RU.ts`
- Test: `frontend/i18n-new-locales.test.mjs`

- [ ] **Step 1: Register the locale modules**

Add imports and messages in `frontend/src/i18n/index.ts`:

```ts
import jaJP from './locales/ja-JP.ts'
import viVN from './locales/vi-VN.ts'

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ru-RU': ruRU,
  'ko-KR': koKR,
  'ja-JP': jaJP,
  'vi-VN': viVN,
}
```

Validate the saved value before passing it to vue-i18n so an obsolete/invalid value cannot select an unregistered locale:

```ts
const supportedLocales = Object.keys(messages)
const storedLocale = localStorage.getItem('locale') || ''
const savedLocale = supportedLocales.includes(storedLocale) ? storedLocale : 'zh-CN'
```

- [ ] **Step 2: Add translated language-name keys to all six bundles**

Under each bundle's existing `language` object, add the same two keys:

```ts
jaJP: '日本語',
viVN: 'Tiếng Việt',
```

In `ja-JP.ts` and `vi-VN.ts`, these keys must occupy the same paths so Task 1 parity remains exact.

- [ ] **Step 3: Add settings options**

Append to the existing `<t-select>` in `GeneralSettings.vue`:

```vue
<t-option value="ja-JP" :label="$t('language.jaJP')">{{ $t('language.jaJP') }}</t-option>
<t-option value="vi-VN" :label="$t('language.viVN')">{{ $t('language.viVN') }}</t-option>
```

- [ ] **Step 4: Add pre-login options**

Append to `Login.vue`'s `languageOptions`:

```ts
{ value: 'ja-JP', label: '日本語', shortLabel: '日本語', flag: '🇯🇵' },
{ value: 'vi-VN', label: 'Tiếng Việt', shortLabel: 'VI', flag: '🇻🇳' },
```

- [ ] **Step 5: Add embed-channel default-language options**

Append to `AgentEmbedChannelPanel.vue`:

```ts
{ label: '日本語', value: 'ja-JP' },
{ label: 'Tiếng Việt', value: 'vi-VN' },
```

- [ ] **Step 6: Run the focused test**

Run: `cd frontend && node --test i18n-new-locales.test.mjs`

Expected: main-bundle parity passes; registration test still identifies `embed.ts` until Task 4.

- [ ] **Step 7: Commit main-app registration**

```bash
git add frontend/src/i18n/index.ts frontend/src/views/settings/GeneralSettings.vue frontend/src/views/auth/Login.vue frontend/src/components/AgentEmbedChannelPanel.vue frontend/src/i18n/locales/*.ts
git commit -m "feat: register Japanese and Vietnamese UI locales"
```

### Task 4: Add Japanese and Vietnamese embedded-chat locales

**Files:**
- Modify: `frontend/src/i18n/embed.ts`
- Modify: `frontend/i18n-new-locales.test.mjs`

- [ ] **Step 1: Extend the failing test with normalization assertions**

Add source-contract assertions that require both prefixes and both message entries:

```js
test('embed locale normalization covers Japanese and Vietnamese prefixes', () => {
  const source = readFileSync(new URL('./src/i18n/embed.ts', import.meta.url), 'utf8')
  assert.match(source, /startsWith\('ja'\).*return 'ja-JP'/)
  assert.match(source, /startsWith\('vi'\).*return 'vi-VN'/)
  assert.match(source, /'ja-JP': deepMerge/)
  assert.match(source, /'vi-VN': deepMerge/)
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd frontend && node --test i18n-new-locales.test.mjs`

Expected: FAIL because `normalizeEmbedLocale` and embed messages do not yet cover either tag.

- [ ] **Step 3: Add complete visitor-message patches**

In `embed.ts`, add `jaEmbedMessages` and `viEmbedMessages` with the same recursive leaf paths as `messages['en-US']`. Translate all visitor chat, input, upload, error, status, and embed-publishing messages; preserve tokens and merge them over English:

```ts
const jaEmbedMessages = {
  embedPublish: {
    title: 'ウェブ埋め込み',
  },
  input: {
    placeholder: 'モデルに質問してください',
  },
} as const

const viEmbedMessages = {
  embedPublish: {
    title: 'Nhúng vào trang web',
  },
  input: {
    placeholder: 'Đặt câu hỏi cho mô hình',
  },
} as const
```

These snippets define the required merge shape. Before registration, compare `leafEntries(jaEmbedMessages)` and `leafEntries(viEmbedMessages)` to `leafEntries(messages['en-US'])` with the same token check used for main bundles; both path sets must be identical.

- [ ] **Step 4: Register and normalize both embed locales**

Update the supported list and prefix handling:

```ts
const SUPPORTED_LOCALES = ['zh-CN', 'en-US', 'ko-KR', 'ru-RU', 'ja-JP', 'vi-VN'] as const

if (s.startsWith('ja')) return 'ja-JP'
if (s.startsWith('vi')) return 'vi-VN'
```

Add both messages to `createI18n`:

```ts
'ja-JP': deepMerge(messages['en-US'], jaEmbedMessages),
'vi-VN': deepMerge(messages['en-US'], viEmbedMessages),
```

- [ ] **Step 5: Run the focused tests**

Run: `cd frontend && node --test i18n-new-locales.test.mjs`

Expected: all tests PASS.

- [ ] **Step 6: Commit embed support**

```bash
git add frontend/src/i18n/embed.ts frontend/i18n-new-locales.test.mjs
git commit -m "feat: localize embedded chat in Japanese and Vietnamese"
```

### Task 5: Compile and verify both delivery surfaces

**Files:**
- Test: `frontend/i18n-new-locales.test.mjs`
- Verify: `frontend/src/i18n/locales/ja-JP.ts`
- Verify: `frontend/src/i18n/locales/vi-VN.ts`

- [ ] **Step 1: Run all frontend Node tests**

Run: `cd frontend && npm test`

Expected: PASS with no locale parity, registration, or existing frontend test failures.

- [ ] **Step 2: Run TypeScript validation**

Run: `cd frontend && npm run type-check`

Expected: exit code 0.

- [ ] **Step 3: Build the public/server-hosted frontend**

Run: `cd frontend && npm run build`

Expected: exit code 0 and generated assets under `frontend/dist`.

- [ ] **Step 4: Build the internal Knavis frontend profile**

Run: `$env:VITE_BRAND_PROFILE='internal'; cd frontend; npm run build`

Expected: exit code 0; generated assets contain Japanese and Vietnamese locale strings while retaining Knavis branding.

- [ ] **Step 5: Run desktop-client regression tests**

Run: `go test ./cmd/desktop`

Expected: PASS; desktop packaging/update behavior is unchanged.

- [ ] **Step 6: Manually verify the server-hosted UI**

Start the existing frontend development server and inspect `/login`, the general settings page, a chat page, and an embed page. For both `ja-JP` and `vi-VN`, verify translated text, persistence after reload, no raw message keys, and no horizontal overflow at 1440x900 and 390x844.

- [ ] **Step 7: Commit any verification-only test adjustments**

When validation exposes a missing registration assertion, update only `frontend/i18n-new-locales.test.mjs`, rerun Steps 1-5, then commit:

```bash
git add frontend/i18n-new-locales.test.mjs
git commit -m "test: harden Japanese and Vietnamese locale coverage"
```

Do not commit generated `frontend/dist` output unless the repository's existing release workflow explicitly tracks it.
