import { createI18n } from 'vue-i18n';
import translationEn from '@/locales/en-US/translation.json';
import translationZh from '@/locales/zh-CN/translation.json';

export const LEGACY_LANGUAGE_STORAGE_KEY = 'aigateway-console.language';
export const LANGUAGE_STORAGE_KEY = 'aigateway-console.language';

export const lngs = [
  {
    code: 'zh-CN',
    nativeName: '简体中文',
    switchText: '中',
    officialSiteCode: 'zh-cn',
  },
  {
    code: 'en-US',
    nativeName: 'English (United States)',
    switchText: 'En',
    officialSiteCode: 'en-us',
  },
] as const;

function resolveLocale() {
  const stored = localStorage.getItem(LANGUAGE_STORAGE_KEY);
  if (stored === 'zh-CN' || stored === 'en-US') {
    return stored;
  }
  const legacyStored = localStorage.getItem(LEGACY_LANGUAGE_STORAGE_KEY);
  if (legacyStored === 'zh-CN' || legacyStored === 'en-US') {
    localStorage.setItem(LANGUAGE_STORAGE_KEY, legacyStored);
    localStorage.removeItem(LEGACY_LANGUAGE_STORAGE_KEY);
    return legacyStored;
  }
  return navigator.language === 'en-US' ? 'en-US' : 'zh-CN';
}

const i18n = createI18n({
  legacy: false,
  locale: resolveLocale(),
  fallbackLocale: 'zh-CN',
  messages: {
    'en-US': translationEn,
    'zh-CN': translationZh,
  },
});

export default i18n;
