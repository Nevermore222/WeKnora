import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN.ts'
import ruRU from './locales/ru-RU.ts'
import enUS from './locales/en-US.ts'
import koKR from './locales/ko-KR.ts'
import jaJP from './locales/ja-JP.ts'
import viVN from './locales/vi-VN.ts'
import { localizeBrandText } from '@/config/brandName'

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ru-RU': ruRU,
  'ko-KR': koKR,
  'ja-JP': jaJP,
  'vi-VN': viVN
}

// Получаем сохраненный язык из localStorage или используем китайский по умолчанию
const supportedLocales = Object.keys(messages)
const storedLocale = localStorage.getItem('locale') || ''
const savedLocale = supportedLocales.includes(storedLocale) ? storedLocale : 'zh-CN'

const i18n = createI18n({
  legacy: false,
  locale: savedLocale,
  fallbackLocale: 'zh-CN',
  globalInjection: true,
  postTranslation: (translated) =>
    typeof translated === 'string' ? localizeBrandText(translated) : translated,
  // Some translations intentionally embed `<strong>` markup (e.g. agent step summaries).
  // We render them via v-html with our own sanitization, so silence vue-i18n's HTML warning
  // to avoid flooding the console and slowing renders during history loads.
  warnHtmlMessage: false,
  messages
})

export default i18n
