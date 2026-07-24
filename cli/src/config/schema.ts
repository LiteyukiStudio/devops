import type { LunaConfigDocument } from '../commands/types.js'

import { z } from 'zod'

export const OUTPUT_FORMATS = [
  'table',
  'json',
  'raw-json',
  'yaml',
  'jsonl',
  'name',
] as const

const userSnapshotSchema = z
  .object({
    id: z.string().min(1),
    name: z.string().min(1).optional(),
  })
  .passthrough()

const credentialBaseSchema = z.object({
  scopes: z.array(z.string().min(1)).default([]),
  user: userSnapshotSchema.optional(),
  expiresAt: z.iso.datetime().optional(),
  createdAt: z.iso.datetime().optional(),
})

export const oauthCredentialSchema = credentialBaseSchema
  .extend({
    type: z.literal('oauth'),
    accessToken: z.string().min(1),
    refreshToken: z.string().min(1).optional(),
    tokenType: z.string().min(1).optional(),
  })
  .passthrough()

export const accessTokenCredentialSchema = credentialBaseSchema
  .extend({
    type: z.literal('access_token'),
    token: z.string().min(1),
  })
  .passthrough()

export const credentialSchema = z.discriminatedUnion('type', [
  oauthCredentialSchema,
  accessTokenCredentialSchema,
])

export const instanceSchema = z
  .object({
    server: z.string().min(1),
    tls: z
      .object({
        caFile: z.string().default(''),
        insecureSkipVerify: z.boolean().default(false),
      })
      .passthrough()
      .default({ caFile: '', insecureSkipVerify: false }),
    network: z
      .object({
        proxy: z.string().default(''),
        noProxy: z.string().default(''),
      })
      .passthrough()
      .default({ proxy: '', noProxy: '' }),
  })
  .passthrough()

export const projectSnapshotSchema = z
  .object({
    id: z.string().min(1),
    name: z.string().min(1).optional(),
    identifier: z.string().min(1).optional(),
  })
  .passthrough()

export const contextSchema = z
  .object({
    instance: z.string().min(1),
    credential: z.string().min(1).optional(),
    project: projectSnapshotSchema.nullish(),
    language: z.string().default(''),
    output: z.union([z.enum(OUTPUT_FORMATS), z.literal('')]).default(''),
  })
  .passthrough()

export const configDocumentSchema = z
  .object({
    version: z.literal(1),
    currentContext: z.string().min(1).nullable().optional(),
    instances: z.record(z.string().min(1), instanceSchema),
    credentials: z.record(z.string().min(1), credentialSchema),
    contexts: z.record(z.string().min(1), contextSchema),
  })
  .superRefine((document, context) => {
    if (
      document.currentContext
      && !Object.hasOwn(document.contexts, document.currentContext)
    ) {
      context.addIssue({
        code: 'custom',
        path: ['currentContext'],
        message: `Unknown current context "${document.currentContext}".`,
      })
    }

    for (const [name, value] of Object.entries(document.contexts)) {
      if (!Object.hasOwn(document.instances, value.instance)) {
        context.addIssue({
          code: 'custom',
          path: ['contexts', name, 'instance'],
          message: `Context "${name}" references an unknown instance.`,
        })
      }
      if (value.credential && !Object.hasOwn(document.credentials, value.credential)) {
        context.addIssue({
          code: 'custom',
          path: ['contexts', name, 'credential'],
          message: `Context "${name}" references an unknown credential.`,
        })
      }
    }
  })

export type LunaCredential = z.infer<typeof credentialSchema>
export type OAuthCredential = z.infer<typeof oauthCredentialSchema>
export type AccessTokenCredential = z.infer<typeof accessTokenCredentialSchema>
export type StoredLunaConfig = z.infer<typeof configDocumentSchema>

export function emptyConfigDocument(): StoredLunaConfig {
  return {
    version: 1,
    currentContext: null,
    instances: {},
    credentials: {},
    contexts: {},
  }
}

export function parseConfigDocument(value: unknown): StoredLunaConfig {
  return configDocumentSchema.parse(value)
}

export function cloneConfigDocument(config: LunaConfigDocument): StoredLunaConfig {
  return parseConfigDocument(structuredClone(config))
}
