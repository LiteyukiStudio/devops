import type { ApiExecutionRequest, CommandExecutionGlobals } from '../../src/commands/index.js'
import { describe, expect, it } from 'vitest'
import {

  CliCommandError,

  normalizeMetadata,
  planOpenApiRequest,
} from '../../src/commands/index.js'

const globals: CommandExecutionGlobals = {
  project: 'project alpha',
  output: 'json',
  color: false,
  interactive: false,
  yes: false,
  quiet: true,
  agent: true,
  timeoutMs: 30_000,
  debug: false,
  insecureSkipTlsVerify: false,
}

describe('openAPI request planning', () => {
  it('maps path, query, headers, body and server dry-run', () => {
    const request: ApiExecutionRequest = {
      operationId: 'updateApplication',
      globals: { ...globals, dryRun: 'server' },
      params: {
        applicationId: 'app/one',
        trace: 'trace-1',
        body: { name: 'demo' },
      },
      metadata: normalizeMetadata({
        category: 'application',
        tool: 'update',
        source: 'openapi',
        operationId: 'updateApplication',
        method: 'patch',
        path: '/api/v1/projects/{projectId}/applications/{applicationId}',
        parameters: [
          { name: 'projectId', location: 'path', required: true },
          { name: 'applicationId', location: 'path', required: true },
          { name: 'trace', location: 'header' },
          { name: 'body', location: 'body' },
        ],
      }),
    }

    expect(planOpenApiRequest(request)).toEqual({
      method: 'PATCH',
      path: '/api/v1/projects/project%20alpha/applications/app%2Fone',
      query: { dryRun: true },
      headers: { trace: 'trace-1' },
      body: { name: 'demo' },
    })
  })

  it('rejects header injection', () => {
    const request: ApiExecutionRequest = {
      operationId: 'inspect',
      globals,
      params: { trace: 'ok\r\nx-unsafe: yes' },
      metadata: normalizeMetadata({
        category: 'api',
        tool: 'inspect',
        source: 'openapi',
        operationId: 'inspect',
        method: 'get',
        path: '/api/v1/inspect',
        parameters: [{ name: 'trace', location: 'header' }],
      }),
    }

    expect(() => planOpenApiRequest(request)).toThrowError(CliCommandError)
  })
})
