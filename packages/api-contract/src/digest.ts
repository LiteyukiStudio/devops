export type JsonPrimitive = string | number | boolean | null;
export type JsonValue =
  | JsonPrimitive
  | readonly JsonValue[]
  | { readonly [key: string]: JsonValue };

const SHA256_INITIAL_STATE = [
  0x6a09e667,
  0xbb67ae85,
  0x3c6ef372,
  0xa54ff53a,
  0x510e527f,
  0x9b05688c,
  0x1f83d9ab,
  0x5be0cd19,
] as const;

const SHA256_ROUND_CONSTANTS = [
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b,
  0x59f111f1, 0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01,
  0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7,
  0xc19bf174, 0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
  0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152,
  0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147,
  0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc,
  0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819,
  0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116, 0x1e376c08,
  0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f,
  0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
  0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
] as const;

function rotateRight(value: number, amount: number): number {
  return (value >>> amount) | (value << (32 - amount));
}

function encodeUtf8(value: string): number[] {
  const bytes: number[] = [];

  for (let index = 0; index < value.length; index += 1) {
    let codePoint = value.charCodeAt(index);

    if (
      codePoint >= 0xd800 &&
      codePoint <= 0xdbff &&
      index + 1 < value.length
    ) {
      const trailing = value.charCodeAt(index + 1);
      if (trailing >= 0xdc00 && trailing <= 0xdfff) {
        codePoint =
          0x10000 + ((codePoint - 0xd800) << 10) + (trailing - 0xdc00);
        index += 1;
      }
    }

    if (codePoint >= 0xd800 && codePoint <= 0xdfff) {
      codePoint = 0xfffd;
    }

    if (codePoint < 0x80) {
      bytes.push(codePoint);
    } else if (codePoint < 0x800) {
      bytes.push(0xc0 | (codePoint >>> 6), 0x80 | (codePoint & 0x3f));
    } else if (codePoint < 0x10000) {
      bytes.push(
        0xe0 | (codePoint >>> 12),
        0x80 | ((codePoint >>> 6) & 0x3f),
        0x80 | (codePoint & 0x3f),
      );
    } else {
      bytes.push(
        0xf0 | (codePoint >>> 18),
        0x80 | ((codePoint >>> 12) & 0x3f),
        0x80 | ((codePoint >>> 6) & 0x3f),
        0x80 | (codePoint & 0x3f),
      );
    }
  }

  return bytes;
}

/**
 * Computes SHA-256 without relying on Node.js or browser globals.
 */
export function sha256Hex(value: string): string {
  const bytes = encodeUtf8(value);
  const bitLength = bytes.length * 8;
  const highLength = Math.floor(bitLength / 0x1_0000_0000);
  const lowLength = bitLength >>> 0;

  bytes.push(0x80);
  while (bytes.length % 64 !== 56) {
    bytes.push(0);
  }

  bytes.push(
    (highLength >>> 24) & 0xff,
    (highLength >>> 16) & 0xff,
    (highLength >>> 8) & 0xff,
    highLength & 0xff,
    (lowLength >>> 24) & 0xff,
    (lowLength >>> 16) & 0xff,
    (lowLength >>> 8) & 0xff,
    lowLength & 0xff,
  );

  const state: number[] = [...SHA256_INITIAL_STATE];
  const schedule = new Array<number>(64).fill(0);

  for (let offset = 0; offset < bytes.length; offset += 64) {
    for (let index = 0; index < 16; index += 1) {
      const position = offset + index * 4;
      schedule[index] =
        ((bytes[position] ?? 0) << 24) |
        ((bytes[position + 1] ?? 0) << 16) |
        ((bytes[position + 2] ?? 0) << 8) |
        (bytes[position + 3] ?? 0);
    }

    for (let index = 16; index < 64; index += 1) {
      const first =
        rotateRight(schedule[index - 15] ?? 0, 7) ^
        rotateRight(schedule[index - 15] ?? 0, 18) ^
        ((schedule[index - 15] ?? 0) >>> 3);
      const second =
        rotateRight(schedule[index - 2] ?? 0, 17) ^
        rotateRight(schedule[index - 2] ?? 0, 19) ^
        ((schedule[index - 2] ?? 0) >>> 10);
      schedule[index] =
        ((schedule[index - 16] ?? 0) +
          first +
          (schedule[index - 7] ?? 0) +
          second) >>>
        0;
    }

    let [a, b, c, d, e, f, g, h] = state;

    for (let index = 0; index < 64; index += 1) {
      const sigmaOne =
        rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25);
      const choose = (e & f) ^ (~e & g);
      const temporaryOne =
        (h +
          sigmaOne +
          choose +
          (SHA256_ROUND_CONSTANTS[index] ?? 0) +
          (schedule[index] ?? 0)) >>>
        0;
      const sigmaZero =
        rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22);
      const majority = (a & b) ^ (a & c) ^ (b & c);
      const temporaryTwo = (sigmaZero + majority) >>> 0;

      h = g;
      g = f;
      f = e;
      e = (d + temporaryOne) >>> 0;
      d = c;
      c = b;
      b = a;
      a = (temporaryOne + temporaryTwo) >>> 0;
    }

    state[0] = (state[0] + a) >>> 0;
    state[1] = (state[1] + b) >>> 0;
    state[2] = (state[2] + c) >>> 0;
    state[3] = (state[3] + d) >>> 0;
    state[4] = (state[4] + e) >>> 0;
    state[5] = (state[5] + f) >>> 0;
    state[6] = (state[6] + g) >>> 0;
    state[7] = (state[7] + h) >>> 0;
  }

  return state.map((part) => part.toString(16).padStart(8, "0")).join("");
}

function serializeJson(
  value: unknown,
  ancestors: Set<object>,
  inArray: boolean,
): string | undefined {
  if (value === null) {
    return "null";
  }

  switch (typeof value) {
    case "string":
    case "boolean":
      return JSON.stringify(value);
    case "number":
      if (!Number.isFinite(value)) {
        throw new TypeError("Cannot digest non-finite numbers");
      }
      return JSON.stringify(value);
    case "undefined":
      return inArray ? "null" : undefined;
    case "bigint":
    case "function":
    case "symbol":
      throw new TypeError(`Cannot digest values of type ${typeof value}`);
    case "object":
      break;
    default:
      throw new TypeError("Cannot digest unsupported value");
  }

  const object = value as object;
  if (ancestors.has(object)) {
    throw new TypeError("Cannot digest cyclic values");
  }

  ancestors.add(object);
  try {
    if (Array.isArray(value)) {
      return `[${value
        .map((item) => serializeJson(item, ancestors, true) ?? "null")
        .join(",")}]`;
    }

    const entries = Object.keys(value as Record<string, unknown>)
      .sort()
      .flatMap((key) => {
        const serialized = serializeJson(
          (value as Record<string, unknown>)[key],
          ancestors,
          false,
        );
        return serialized === undefined
          ? []
          : [`${JSON.stringify(key)}:${serialized}`];
      });
    return `{${entries.join(",")}}`;
  } finally {
    ancestors.delete(object);
  }
}

/**
 * Serializes JSON-compatible data with recursively sorted object keys.
 */
export function stableStringify(value: unknown): string {
  return serializeJson(value, new Set<object>(), false) ?? "null";
}

export function digestJson(value: unknown): `sha256:${string}` {
  return `sha256:${sha256Hex(stableStringify(value))}`;
}
