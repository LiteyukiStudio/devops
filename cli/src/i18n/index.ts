import type { i18n } from 'i18next'
import type { LocaleDetectionOptions } from './locale.js'
import type { SupportedLocale } from './resources.js'
import { createInstance } from 'i18next'
import { detectLocale } from './locale.js'
import { resources } from './resources.js'

export interface CreateI18nOptions extends LocaleDetectionOptions {
  readonly fallbackLocale?: SupportedLocale
}

export async function createCliI18n(options: CreateI18nOptions = {}): Promise<i18n> {
  const instance = createInstance()
  await instance.init({
    lng: detectLocale(options),
    fallbackLng: options.fallbackLocale ?? 'en-US',
    supportedLngs: Object.keys(resources),
    resources,
    interpolation: { escapeValue: false },
    returnNull: false,
  })
  return instance
}

export function assertLocaleResourceParity(): void {
  const reference = flattenKeys(resources['en-US'].translation)
  const referenceValues = flattenValues(resources['en-US'].translation)
  for (const [locale, bundle] of Object.entries(resources)) {
    const current = flattenKeys(bundle.translation)
    const missing = reference.filter(key => !current.includes(key))
    const extra = current.filter(key => !reference.includes(key))
    const currentValues = flattenValues(bundle.translation)
    const placeholderMismatch = reference.filter((key) => {
      const expected = placeholders(referenceValues[key] ?? '')
      const actual = placeholders(currentValues[key] ?? '')
      return expected.join(',') !== actual.join(',')
    })
    if (missing.length > 0 || extra.length > 0 || placeholderMismatch.length > 0) {
      throw new Error(
        `Locale ${locale} is inconsistent: missing=${missing.join(',')} extra=${extra.join(',')} placeholders=${placeholderMismatch.join(',')}`,
      )
    }
  }
}

function flattenKeys(value: object, prefix = ''): string[] {
  return Object.entries(value).flatMap(([key, child]) => {
    const path = prefix ? `${prefix}.${key}` : key
    return typeof child === 'object' && child !== null
      ? flattenKeys(child, path)
      : [path]
  })
}

function flattenValues(value: object, prefix = ''): Record<string, string> {
  return Object.fromEntries(Object.entries(value).flatMap(([key, child]) => {
    const path = prefix ? `${prefix}.${key}` : key
    return typeof child === 'object' && child !== null
      ? Object.entries(flattenValues(child, path))
      : [[path, String(child)]]
  }))
}

function placeholders(value: string): string[] {
  return [...value.matchAll(/\{\{\s*([\w.]+)(?:\s*,[^}]*)?\}\}/gu)]
    .map(match => match[1]!)
    .sort()
}

export * from './locale.js'
export * from './resources.js'
