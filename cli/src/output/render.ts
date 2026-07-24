import { stringify as stringifyYaml } from "yaml";
import { redactValue, sanitizeTerminalText } from "../errors/index.js";
import { stringifyJson } from "./json.js";

export interface TableColumn<T extends Record<string, unknown> = Record<string, unknown>> {
  readonly key: keyof T & string;
  readonly header?: string;
  readonly render?: (value: unknown, row: T) => unknown;
}

export interface HumanRenderOptions<T extends Record<string, unknown> = Record<string, unknown>> {
  readonly columns?: readonly TableColumn<T>[];
  readonly emptyText?: string;
}

export function renderTable<T extends Record<string, unknown>>(
  rows: readonly T[],
  options: HumanRenderOptions<T> = {},
): string {
  if (rows.length === 0) return sanitizeTerminalText(options.emptyText ?? "");
  const columns = options.columns ?? inferColumns(rows);
  if (columns.length === 0) return "";

  const matrix = [
    columns.map(column => sanitizeTerminalText(column.header ?? column.key)),
    ...rows.map(row => columns.map(column =>
      formatCell(column.render ? column.render(row[column.key], row) : row[column.key]))),
  ];
  const widths = columns.map((_, columnIndex) =>
    Math.max(...matrix.map(row => displayWidth(row[columnIndex] ?? ""))));

  return matrix
    .map((row) => {
      const cells = row.map((cell, columnIndex) =>
        padDisplay(cell, widths[columnIndex]!));
      return cells.join("  ").trimEnd();
    })
    .join("\n");
}

export function renderFieldView(
  value: Readonly<Record<string, unknown>>,
  labels: Readonly<Record<string, string>> = {},
): string {
  const safe = redactValue(value) as Readonly<Record<string, unknown>>;
  const entries = Object.entries(safe);
  if (entries.length === 0) return "";
  const width = Math.max(...entries.map(([key]) => displayWidth(labels[key] ?? key)));
  return entries
    .map(([key, fieldValue]) =>
      `${padDisplay(sanitizeTerminalText(labels[key] ?? key), width)}  ${formatCell(fieldValue)}`)
    .join("\n");
}

export function renderHuman(
  value: unknown,
  options: HumanRenderOptions = {},
): string {
  if (Array.isArray(value)) {
    const rows = value.filter(isRecord);
    return rows.length === value.length
      ? renderTable(rows, options)
      : value.map(formatCell).join("\n");
  }
  if (isRecord(value)) return renderFieldView(value);
  return formatCell(value);
}

export function renderYaml(value: unknown): string {
  return stringifyYaml(redactValue(value), {
    lineWidth: 0,
    sortMapEntries: false,
  }).trimEnd();
}

export function renderNames(value: unknown): string {
  if (Array.isArray(value)) return value.map(extractName).filter(Boolean).join("\n");
  return extractName(value);
}

function inferColumns<T extends Record<string, unknown>>(
  rows: readonly T[],
): readonly TableColumn<T>[] {
  const keys = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row)) keys.add(key);
  }
  return [...keys].map(key => ({ key: key as keyof T & string }));
}

function extractName(value: unknown): string {
  if (isRecord(value)) {
    for (const key of ["name", "id", "identifier"]) {
      if (typeof value[key] === "string") return sanitizeTerminalText(value[key]);
    }
    return "";
  }
  return typeof value === "string" ? sanitizeTerminalText(value) : "";
}

function formatCell(value: unknown): string {
  const safe = redactValue(value);
  if (safe === null || safe === undefined) return "";
  if (typeof safe === "string") return sanitizeTerminalText(safe).replace(/[\r\n]+/gu, " ");
  if (typeof safe === "number" || typeof safe === "boolean" || typeof safe === "bigint") {
    return String(safe);
  }
  return stringifyJson(safe);
}

function displayWidth(value: string): number {
  let width = 0;
  for (const character of value) {
    const codePoint = character.codePointAt(0)!;
    width += isWideCodePoint(codePoint) ? 2 : 1;
  }
  return width;
}

function padDisplay(value: string, width: number): string {
  return `${value}${" ".repeat(Math.max(0, width - displayWidth(value)))}`;
}

function isWideCodePoint(codePoint: number): boolean {
  return codePoint >= 0x1100 && (
    codePoint <= 0x115f
    || codePoint === 0x2329
    || codePoint === 0x232a
    || (codePoint >= 0x2e80 && codePoint <= 0xa4cf && codePoint !== 0x303f)
    || (codePoint >= 0xac00 && codePoint <= 0xd7a3)
    || (codePoint >= 0xf900 && codePoint <= 0xfaff)
    || (codePoint >= 0xfe10 && codePoint <= 0xfe19)
    || (codePoint >= 0xfe30 && codePoint <= 0xfe6f)
    || (codePoint >= 0xff00 && codePoint <= 0xff60)
    || (codePoint >= 0xffe0 && codePoint <= 0xffe6)
    || (codePoint >= 0x1f300 && codePoint <= 0x1faff)
    || (codePoint >= 0x20000 && codePoint <= 0x3fffd)
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
