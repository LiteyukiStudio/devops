export type InputValueType = 'string' | 'boolean' | 'integer' | 'number' | 'array' | 'object' | 'binary' | 'null'
export type InputSourceKind = 'inline' | 'file' | 'stdin'

export interface InputSchema {
  readonly 'type'?: InputValueType | readonly InputValueType[]
  readonly 'format'?: string
  readonly 'enum'?: readonly unknown[]
  readonly 'properties'?: Readonly<Record<string, InputSchema>>
  readonly 'required'?: readonly string[]
  readonly 'items'?: InputSchema
  readonly 'additionalProperties'?: boolean | InputSchema
  readonly 'minLength'?: number
  readonly 'maxLength'?: number
  readonly 'minimum'?: number
  readonly 'maximum'?: number
  readonly 'minItems'?: number
  readonly 'maxItems'?: number
  readonly 'writeOnly'?: boolean
  readonly 'x-sensitive'?: boolean
  readonly [key: string]: unknown
}

export interface InputField {
  readonly name: string
  readonly schema?: InputSchema
  readonly required?: boolean
  readonly repeated?: boolean
  readonly sensitive?: boolean
  readonly valueSources?: readonly InputSourceKind[]
}

export interface CommandInputSpec {
  readonly command?: string
  readonly fields: readonly InputField[]
  readonly paramsSchema?: InputSchema
}

export interface InputLimits {
  readonly inlineBytes: number
  readonly fileBytes: number
  readonly stdinBytes: number
  readonly paramsBytes: number
}

export interface InputSourceReader {
  readFile: (path: string, maxBytes: number) => Promise<Uint8Array>
  readStdin: (maxBytes: number) => Promise<Uint8Array>
}

export interface ParsedValueSource {
  readonly kind: InputSourceKind
  readonly inlineValue?: string
  readonly path?: string
}

export interface ParsedCommandInput {
  readonly values: Readonly<Record<string, unknown>>
  readonly stdinUsed: boolean
}

export const DEFAULT_INPUT_LIMITS: InputLimits = Object.freeze({
  inlineBytes: 4 * 1024,
  fileBytes: 10 * 1024 * 1024,
  stdinBytes: 10 * 1024 * 1024,
  paramsBytes: 1024 * 1024,
})
