import { describe, expect, it } from 'vitest'
import {
  parseBoolean,
  parseDurationMilliseconds,
  parseInteger,
  parseNumberValue,
} from '../../src/input/index.js'

describe('input primitives', () => {
  it('parses strict booleans and finite numbers', () => {
    expect(parseBoolean('true')).toBe(true)
    expect(parseBoolean('false')).toBe(false)
    expect(parseInteger('-42')).toBe(-42)
    expect(parseNumberValue('1.25e2')).toBe(125)
  })

  it('rejects ambiguous booleans and unsafe integers', () => {
    expect(() => parseBoolean('yes')).toThrowError(/true.*false/u)
    expect(() => parseInteger('9007199254740992')).toThrowError(/safe integer/u)
  })

  it('parses Go-style combined durations', () => {
    expect(parseDurationMilliseconds('1h30m5.5s')).toBe(5_405_500)
    expect(parseDurationMilliseconds('-250ms')).toBe(-250)
    expect(parseDurationMilliseconds('0')).toBe(0)
    expect(() => parseDurationMilliseconds('5 minutes')).toThrowError(/Go duration/u)
  })
})
