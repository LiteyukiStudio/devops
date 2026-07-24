import { escapeUnsafeJsonCharacters, redactValue } from "../errors/index.js";

export interface JsonStringifyOptions {
  readonly pretty?: boolean;
  readonly redact?: boolean;
}

export function stringifyJson(value: unknown, options: JsonStringifyOptions = {}): string {
  const safeValue = options.redact === false ? value : redactValue(value);
  const serialized = JSON.stringify(safeValue, null, options.pretty ? 2 : undefined);
  return escapeUnsafeJsonCharacters(serialized ?? "null");
}

export function stringifyJsonLine(value: unknown): string {
  return `${stringifyJson(value)}\n`;
}
