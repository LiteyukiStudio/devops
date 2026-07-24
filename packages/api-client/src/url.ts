import type { QueryInput, QueryPrimitive } from "./types.js"

function serializeQueryPrimitive(value: QueryPrimitive): string {
  if (value instanceof Date) {
    return value.toISOString()
  }
  return String(value)
}

export function normalizeBaseUrl(input: string | URL): URL {
  const url = new URL(input)
  if (url.username || url.password) {
    throw new TypeError("baseUrl must not contain credentials")
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new TypeError("baseUrl must use http or https")
  }
  url.hash = ""
  return url
}

export function resolveRequestUrl(
  baseUrl: URL,
  path: string | URL,
  query?: QueryInput,
): URL {
  const url = path instanceof URL ? new URL(path) : new URL(path, baseUrl)
  if (url.username || url.password) {
    throw new TypeError("request URL must not contain credentials")
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new TypeError("request URL must use http or https")
  }
  url.hash = ""

  if (query instanceof URLSearchParams) {
    for (const [key, value] of query) {
      url.searchParams.append(key, value)
    }
    return url
  }

  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === null || value === undefined) {
        continue
      }
      const values = Array.isArray(value) ? value : [value]
      for (const item of values) {
        url.searchParams.append(key, serializeQueryPrimitive(item))
      }
    }
  }
  return url
}

export function sameOrigin(left: URL, right: URL): boolean {
  return left.origin === right.origin
}

