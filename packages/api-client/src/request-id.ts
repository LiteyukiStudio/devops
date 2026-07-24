export function createRequestId(): string {
  const randomUUID = globalThis.crypto?.randomUUID
  if (typeof randomUUID === "function") {
    return `req_${randomUUID.call(globalThis.crypto)}`
  }
  const random = Math.random().toString(36).slice(2)
  return `req_${Date.now().toString(36)}_${random}`
}

