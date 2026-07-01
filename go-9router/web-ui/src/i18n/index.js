// Simplified i18n runtime — dictionary-based.
// Instead of the MutationObserver DOM-walker used in the original Next.js app,
// this version exposes a `translate(text)` function that components call directly.
// This is more predictable in a CSR SPA and avoids the complexity of the DOM walker.

export const LOCALES = [
  "en", "vi", "zh-CN", "zh-TW", "ja", "pt-BR", "pt-PT", "ko",
  "es", "de", "fr", "he", "ar", "ru", "pl", "cs", "nl", "tr",
  "uk", "tl", "id", "th", "hi", "bn", "ur", "ro", "sv", "it",
  "el", "hu", "fi", "da", "no",
];

export const DEFAULT_LOCALE = "en";
export const LOCALE_COOKIE = "locale";

export const LOCALE_NAMES = {
  en: "English",
  vi: "Tiếng Việt",
  "zh-CN": "简体中文",
  "zh-TW": "繁體中文",
  ja: "日本語",
  "pt-BR": "Português (Brasil)",
  "pt-PT": "Português (Portugal)",
  ko: "한국어",
  es: "Español",
  de: "Deutsch",
  fr: "Français",
  he: "עברית",
  ar: "العربية",
  ru: "Русский",
  pl: "Polski",
  cs: "Čeština",
  nl: "Nederlands",
  tr: "Türkçe",
  uk: "Українська",
  tl: "Tagalog",
  id: "Indonesia",
  th: "ไทย",
  hi: "हिन्दी",
  bn: "বাংলা",
  ur: "اردو",
  ro: "Română",
  sv: "Svenska",
  it: "Italiano",
  el: "Ελληνικά",
  hu: "Magyar",
  fi: "Suomi",
  da: "Dansk",
  no: "Norsk",
};

let translationMap = {};
let currentLocale = DEFAULT_LOCALE;
let changeCallbacks = [];

function normalizeLocale(locale) {
  if (!locale) return DEFAULT_LOCALE;
  if (locale === "zh") return "zh-CN";
  if (LOCALES.includes(locale)) return locale;
  return DEFAULT_LOCALE;
}

function getLocaleFromCookie() {
  if (typeof document === "undefined") return DEFAULT_LOCALE;
  const cookie = document.cookie
    .split(";")
    .find((c) => c.trim().startsWith(`${LOCALE_COOKIE}=`));
  const value = cookie ? decodeURIComponent(cookie.split("=")[1]) : DEFAULT_LOCALE;
  return normalizeLocale(value);
}

async function loadTranslations(locale) {
  if (locale === "en") {
    translationMap = {};
    return;
  }
  try {
    const response = await fetch(`/i18n/literals/${locale}.json`);
    translationMap = await response.json();
  } catch {
    translationMap = {};
  }
}

export function translate(text) {
  if (!text || typeof text !== "string") return text;
  const trimmed = text.trim();
  if (!trimmed) return text;
  if (currentLocale === "en") return text;
  return translationMap[trimmed] || text;
}

export function getCurrentLocale() {
  return currentLocale;
}

export function onLocaleChange(callback) {
  changeCallbacks.push(callback);
  return () => {
    changeCallbacks = changeCallbacks.filter((cb) => cb !== callback);
  };
}

export async function initI18n() {
  if (typeof window === "undefined") return;
  currentLocale = getLocaleFromCookie();
  await loadTranslations(currentLocale);
  // Mark icon font as loaded (material symbols loaded via CSS link)
  document.documentElement.classList.add("fonts-loaded");
}

export async function reloadTranslations() {
  currentLocale = getLocaleFromCookie();
  await loadTranslations(currentLocale);
  changeCallbacks.forEach((cb) => cb());
}
