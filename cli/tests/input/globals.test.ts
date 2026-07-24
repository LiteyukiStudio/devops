import { describe, expect, it } from 'vitest'
import { extractGlobalControls } from '../../src/input/index.js'

describe('global controls', () => {
  it('extracts controls without consuming business arguments', () => {
    const parsed = extractGlobalControls([
      'name=demo',
      'context=prod',
      'output=json',
      'timeout=30s',
      'agent=true',
    ])

    expect(parsed.businessTokens).toEqual(['name=demo'])
    expect(parsed.controls).toEqual({
      context: 'prod',
      output: 'json',
      timeoutMs: 30_000,
      agent: true,
    })
    expect(parsed.explicitKeys).toEqual(new Set(['context', 'output', 'timeout', 'agent']))
  })

  it('rejects duplicate controls and unsupported output formats', () => {
    expect(() => extractGlobalControls(['quiet=true', 'quiet=false']))
      .toThrowError(/only be specified once/u)
    expect(() => extractGlobalControls(['output=xml']))
      .toThrowError(/Unsupported output format/u)
  })
})
