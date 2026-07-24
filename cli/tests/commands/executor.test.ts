import type { CommandExecutionGlobals, CommandResult, LunaConfigDocument, NormalizedCommandMetadata, RuntimePorts } from '../../src/commands/index.js'
import { describe, expect, it } from 'vitest'
import {

  CommandRegistry,

  createCliProgram,
  DefaultInputPort,

  runCli,

} from '../../src/commands/index.js'

const emptyConfig: LunaConfigDocument = {
  version: 1,
  currentContext: null,
  instances: {},
  credentials: {},
  contexts: {},
}

describe('commander command execution', () => {
  it('forces machine-readable agent globals', async () => {
    const registry = new CommandRegistry()
    registry.register({
      category: 'version',
      tool: 'show',
      source: 'local',
    }, async invocation => ({ data: invocation.globals }))
    const captures = capturePorts()
    const program = createCliProgram({ registry, ports: captures.ports })

    const result = await runCli(program, ['node', 'luna', 'version', 'show', '--agent'])

    expect(result.exitCode).toBe(0)
    expect(captures.successes[0]?.globals).toMatchObject({
      agent: true,
      output: 'json',
      color: false,
      interactive: false,
      quiet: true,
    })
  })

  it('rejects an explicit project for a project-independent command', async () => {
    const registry = new CommandRegistry()
    registry.register({
      category: 'version',
      tool: 'show',
      source: 'local',
      projectContext: 'none',
    }, async () => ({ data: {} }))
    const captures = capturePorts()
    const program = createCliProgram({ registry, ports: captures.ports })

    const result = await runCli(
      program,
      ['node', 'luna', 'version', 'show', 'project=project-1'],
      captures.ports.output,
    )

    expect(result.exitCode).toBe(2)
    expect(captures.errors).toHaveLength(1)
    expect((captures.errors[0] as { code?: string }).code).toBe('project_not_supported')
  })

  it('does not treat Commander boolean defaults as explicit conflicts', async () => {
    const registry = new CommandRegistry()
    registry.register({
      category: 'help',
      tool: 'catalog',
      source: 'local',
    }, async invocation => ({ data: invocation.globals }))
    const captures = capturePorts()
    const program = createCliProgram({ registry, ports: captures.ports })

    const result = await runCli(
      program,
      ['node', 'luna', 'help', 'catalog', 'interactive=false', 'color=false'],
      captures.ports.output,
    )

    expect(result.exitCode).toBe(0)
    expect(captures.successes[0]?.globals).toMatchObject({
      interactive: false,
      color: false,
    })
  })

  it('requires confirmation for medium-risk API mutations in non-interactive mode', async () => {
    const registry = new CommandRegistry()
    registry.register({
      category: 'application',
      tool: 'create',
      source: 'openapi',
      operationId: 'createApplication',
      risk: 'medium',
    }, async () => ({ data: {} }))
    const captures = capturePorts()
    const program = createCliProgram({ registry, ports: captures.ports })

    const result = await runCli(
      program,
      ['node', 'luna', 'application', 'create'],
      captures.ports.output,
    )

    expect(result.exitCode).toBe(6)
    expect((captures.errors[0] as { code?: string }).code).toBe('confirmation_required')
  })

  it('allows yes=true for medium-risk API mutations', async () => {
    const registry = new CommandRegistry()
    registry.register({
      category: 'application',
      tool: 'create',
      source: 'openapi',
      operationId: 'createApplication',
      risk: 'medium',
    }, async () => ({ data: { created: true } }))
    const captures = capturePorts()
    const program = createCliProgram({ registry, ports: captures.ports })

    const result = await runCli(
      program,
      ['node', 'luna', 'application', 'create', 'yes=true'],
      captures.ports.output,
    )

    expect(result.exitCode).toBe(0)
    expect(captures.successes).toHaveLength(1)
  })

  it('fails closed for high-risk API commands without a server plan', async () => {
    const registry = new CommandRegistry()
    registry.register({
      category: 'application',
      tool: 'delete',
      source: 'openapi',
      operationId: 'deleteApplication',
      risk: 'high',
    }, async () => ({ data: {} }))
    const captures = capturePorts()
    const program = createCliProgram({ registry, ports: captures.ports })

    const result = await runCli(
      program,
      ['node', 'luna', 'application', 'delete', 'yes=true'],
      captures.ports.output,
    )

    expect(result.exitCode).toBe(6)
    expect((captures.errors[0] as { code?: string }).code).toBe('server_plan_required')
  })

  it('uses the shared prompt for high-risk local commands', async () => {
    const registry = new CommandRegistry()
    registry.register({
      category: 'context',
      tool: 'delete',
      source: 'local',
      risk: 'high',
    }, async () => ({ data: { deleted: true } }))
    const captures = capturePorts()
    let prompted = false
    const program = createCliProgram({
      registry,
      ports: {
        ...captures.ports,
        isTTY: true,
        input: {
          parse: captures.ports.input.parse,
          async confirm() {
            prompted = true
            return true
          },
        },
      },
    })

    const result = await runCli(
      program,
      ['node', 'luna', 'context', 'delete'],
      captures.ports.output,
    )

    expect(result.exitCode).toBe(0)
    expect(prompted).toBe(true)
  })
})

function capturePorts(): {
  ports: RuntimePorts
  successes: Array<{
    metadata: NormalizedCommandMetadata
    result: CommandResult
    globals: CommandExecutionGlobals
  }>
  errors: unknown[]
} {
  const successes: Array<{
    metadata: NormalizedCommandMetadata
    result: CommandResult
    globals: CommandExecutionGlobals
  }> = []
  const errors: unknown[] = []
  return {
    successes,
    errors,
    ports: {
      config: {
        read: async () => emptyConfig,
        write: async () => undefined,
      },
      input: new DefaultInputPort(),
      output: {
        writeSuccess(metadata, result, commandGlobals) {
          successes.push({ metadata, result, globals: commandGlobals })
        },
        writeError(error) {
          errors.push(error)
        },
      },
      api: {
        execute: async () => ({ data: {} }),
        request: async () => ({ data: {} }),
      },
      env: {},
      isTTY: false,
      version: 'test',
      distribution: 'source',
    },
  }
}
