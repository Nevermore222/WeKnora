# Japanese and Vietnamese UI Internationalization

## Goal

Add Japanese (`ja-JP`) and Vietnamese (`vi-VN`) as complete interface languages alongside the existing Simplified Chinese, English, Russian, and Korean locales. The new locales must work in the desktop client and in the web application served by the internal server, including the embedded chat surface.

## Scope

- Add complete `ja-JP` and `vi-VN` message bundles using the existing `vue-i18n` schema.
- Register both locales in the main application locale map.
- Add both locales to the settings language selector and preserve the existing local-storage behavior.
- Extend browser-language normalization and fallback behavior for Japanese and Vietnamese.
- Register both locales in the isolated embed i18n instance and support `?locale=ja-JP` and `?locale=vi-VN`.
- Keep interpolation parameters, HTML fragments, pluralization parameters, and message key shapes identical to the English baseline.
- Add regression tests for locale registration, locale normalization, key parity, and interpolation-token parity.

The server continues to serve the compiled frontend assets. The Go API does not gain a separate translated error-message catalog because the existing four language packs are frontend resources rather than server-side response translations.

## Architecture

The main app remains the source of truth for the settings language selector and the persisted `locale` value. `frontend/src/i18n/index.ts` owns the main locale registry and keeps Simplified Chinese as the fallback. `frontend/src/i18n/embed.ts` maintains its isolated registry and storage key so embedded chat language changes do not overwrite the administrator UI language.

Each new locale is a standalone TypeScript module under `frontend/src/i18n/locales/`. The English bundle is the structural baseline. A locale is considered complete only when its recursive key set and interpolation-token set match that baseline. Existing brand post-processing continues to run for all locales.

## Locale Resolution

The resolution order remains:

1. An explicit user selection saved in the existing storage key.
2. An explicit embed URL `?locale=` value for embedded chat.
3. The browser language, normalized by language prefix (`ja` to `ja-JP`, `vi` to `vi-VN`).
4. The existing default/fallback locale.

Unsupported values continue to fall back without changing the behavior of current locales.

## Translation Rules

- Preserve every existing interpolation token exactly, including tokens such as `{name}`, `{count}`, and `{types}`.
- Preserve trusted HTML fragments used by existing `v-html` messages.
- Preserve message keys even when a translated string is intentionally short.
- Keep product and technical identifiers such as `Knavis`, `Xelora Cloud`, API names, file extensions, and command snippets unchanged unless the existing locale convention translates them.
- Do not silently fall back to English for missing keys; key-parity tests must fail during development.

## Verification

- Run focused locale tests first and confirm they fail before implementation.
- Add recursive key-parity and interpolation-token tests against `en-US`.
- Test main-app locale registration and embed locale normalization for `ja`, `ja-JP`, `vi`, and `vi-VN`.
- Run frontend type checking and production build.
- Run the relevant Go/client tests to confirm the server-delivered client remains unaffected.
- Manually verify the settings page, login page, chat page, and embed page in both new locales at desktop and narrow viewport widths.

## Out of Scope

- Translating Go API response messages independently from the existing frontend locale system.
- Changing the default locale or fallback locale.
- Refactoring existing language bundles or changing the existing four translations.
- Adding a translation-management service or runtime remote translation fetch.
