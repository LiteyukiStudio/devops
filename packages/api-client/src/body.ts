function isAvailable<T>(name: string, value: unknown): value is T {
  return typeof globalThis === "object" && name in globalThis && value !== undefined
}

function isBlob(value: unknown): value is Blob {
  return isAvailable<typeof Blob>("Blob", globalThis.Blob) && value instanceof Blob
}

function isFormData(value: unknown): value is FormData {
  return isAvailable<typeof FormData>("FormData", globalThis.FormData) && value instanceof FormData
}

function isUrlSearchParams(value: unknown): value is URLSearchParams {
  return value instanceof URLSearchParams
}

export interface NormalizedBody {
  body: BodyInit | null
  headers: Headers
}

export function normalizeRequestBody(body: unknown, inputHeaders?: HeadersInit): NormalizedBody {
  const headers = new Headers(inputHeaders)
  if (body === null || body === undefined) {
    return { body: null, headers }
  }

  if (
    typeof body === "string"
    || isBlob(body)
    || isFormData(body)
    || isUrlSearchParams(body)
    || body instanceof ArrayBuffer
    || ArrayBuffer.isView(body)
  ) {
    return { body: body as BodyInit, headers }
  }

  if (!headers.has("content-type")) {
    headers.set("content-type", "application/json")
  }
  return { body: JSON.stringify(body), headers }
}

