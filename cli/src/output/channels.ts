import type { Writable } from "node:stream";
import { normalizeLunaError, sanitizeTerminalText, toErrorDocument } from "../errors/index.js";
import type { OutputFormat } from "../input/index.js";
import { stringifyJson, stringifyJsonLine } from "./json.js";
import { renderHuman, renderNames, renderYaml, type HumanRenderOptions } from "./render.js";

export interface OutputStreams {
  readonly stdout: Pick<Writable, "write">;
  readonly stderr: Pick<Writable, "write">;
}

export interface WriteResultOptions {
  readonly format: OutputFormat;
  readonly rawData?: unknown;
  readonly human?: HumanRenderOptions;
}

export class OutputChannels {
  readonly #streams: OutputStreams;
  readonly #quiet: boolean;

  constructor(
    streams: OutputStreams = { stdout: process.stdout, stderr: process.stderr },
    options: { readonly quiet?: boolean } = {},
  ) {
    this.#streams = streams;
    this.#quiet = options.quiet ?? false;
  }

  writeResult(value: unknown, options: WriteResultOptions): void {
    let rendered: string;
    switch (options.format) {
      case "json":
        rendered = stringifyJson(value, { pretty: false });
        break;
      case "raw-json":
        rendered = stringifyJson(options.rawData ?? value, { pretty: false });
        break;
      case "yaml":
        rendered = renderYaml(value);
        break;
      case "name":
        rendered = renderNames(value);
        break;
      case "jsonl":
        if (!Array.isArray(value)) {
          rendered = stringifyJsonLine(value).trimEnd();
          break;
        }
        rendered = value.map(item => stringifyJson(item)).join("\n");
        break;
      default:
        rendered = renderHuman(value, options.human);
    }
    this.#writeLine(this.#streams.stdout, rendered);
  }

  writeJsonLine(value: unknown): void {
    this.#streams.stdout.write(stringifyJsonLine(value));
  }

  writeInfo(message: string): void {
    if (!this.#quiet) this.#writeLine(this.#streams.stderr, sanitizeTerminalText(message));
  }

  writeWarning(message: string): void {
    if (!this.#quiet) this.#writeLine(this.#streams.stderr, sanitizeTerminalText(message));
  }

  writeDebug(message: string, enabled: boolean): void {
    if (enabled) this.#writeLine(this.#streams.stderr, sanitizeTerminalText(message));
  }

  writeError(error: unknown, machine = false): number {
    const normalized = normalizeLunaError(error);
    const rendered = machine
      ? stringifyJson(toErrorDocument(normalized))
      : `${normalized.code}: ${sanitizeTerminalText(normalized.message)}`;
    this.#writeLine(this.#streams.stderr, rendered);
    return normalized.exitCode;
  }

  #writeLine(stream: Pick<Writable, "write">, value: string): void {
    if (value.length === 0) return;
    stream.write(value.endsWith("\n") ? value : `${value}\n`);
  }
}
