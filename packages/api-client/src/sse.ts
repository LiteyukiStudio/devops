import { createParser, type EventSourceMessage } from "eventsource-parser"

import type { LunaSseEvent } from "./types.js"

export interface ParseSseOptions<T> {
  parseData?: (rawData: string) => T
}

export async function* parseSseStream<T = unknown>(
  body: ReadableStream<Uint8Array>,
  options: ParseSseOptions<T> = {},
): AsyncGenerator<LunaSseEvent<T>> {
  const decoder = new TextDecoder()
  const reader = body.getReader()
  const events: EventSourceMessage[] = []
  let parserError: Error | undefined
  const parser = createParser({
    onError(error) {
      parserError = new Error(`Invalid SSE stream: ${error.message}`)
    },
    onEvent(event) {
      events.push(event)
    },
  })

  const drain = function* (): Generator<LunaSseEvent<T>> {
    while (events.length > 0) {
      const event = events.shift()
      if (!event) {
        continue
      }
      let data: T | string = event.data
      if (options.parseData) {
        data = options.parseData(event.data)
      }
      yield {
        data,
        event: event.event || undefined,
        id: event.id || undefined,
        rawData: event.data,
      }
    }
  }

  try {
    while (true) {
      const chunk = await reader.read()
      if (chunk.done) {
        parser.feed(decoder.decode())
        break
      }
      parser.feed(decoder.decode(chunk.value, { stream: true }))
      if (parserError) {
        throw parserError
      }
      yield* drain()
    }
    if (parserError) {
      throw parserError
    }
    yield* drain()
  }
  finally {
    await reader.cancel().catch(() => undefined)
    reader.releaseLock()
  }
}

