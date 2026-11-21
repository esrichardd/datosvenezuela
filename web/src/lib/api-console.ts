/**
 * Logic of the API console on the home page: which endpoints it offers, how
 * it builds their URLs and how it sends a request and describes the outcome.
 * It has no UI, so it can be tested on its own.
 */

export interface ConsoleEndpoint {
  id: string;
  /** Path with one parameter in braces, e.g. "/v1/states/{code}". */
  path: string;
  description: string;
  /** Value the parameter starts with. */
  example: string;
}

export const consoleEndpoints: ConsoleEndpoint[] = [
  {
    id: "state",
    path: "/v1/states/{code}",
    description: "Un estado por su código",
    example: "VE14",
  },
  {
    id: "bank",
    path: "/v1/banks/{code}",
    description: "Un banco por su código",
    example: "0102",
  },
  {
    id: "phone-prefix",
    path: "/v1/phone-prefixes/{code}",
    description: "Un prefijo telefónico",
    example: "414",
  },
  {
    id: "holidays",
    path: "/v1/holidays/national?year={year}",
    description: "Los feriados nacionales de un año",
    example: "2026",
  },
];

export interface PathParts {
  /** Text before the parameter. */
  prefix: string;
  /** Name of the parameter, without braces. */
  param: string;
  /** Text after the parameter. */
  suffix: string;
}

/** Splits a path around its parameter: "/v1/states/{code}" → prefix, "code", "". */
export function splitPath(path: string): PathParts {
  const match = /^(.*)\{(\w+)\}(.*)$/.exec(path);
  if (!match) return { prefix: path, param: "", suffix: "" };
  const [, prefix = "", param = "", suffix = ""] = match;
  return { prefix, param, suffix };
}

/** Full URL of an endpoint for a parameter value. */
export function buildUrl(baseUrl: string, path: string, value: string): string {
  const { prefix, suffix } = splitPath(path);
  const base = baseUrl.replace(/\/+$/, "");
  return `${base}${prefix}${encodeURIComponent(value.trim())}${suffix}`;
}

export type ConsoleResult =
  | {
      kind: "response";
      status: number;
      statusText: string;
      ok: boolean;
      durationMs: number;
      sizeBytes: number;
      requestId: string | null;
      /** The body, pretty-printed when it is JSON. */
      body: string;
    }
  | { kind: "network-error"; durationMs: number }
  | { kind: "timeout"; durationMs: number };

export interface SendOptions {
  /** Defaults to the global fetch; tests pass a fake. */
  fetch?: typeof fetch;
  /** Defaults to performance.now; tests pass a fake clock. */
  now?: () => number;
  timeoutMs?: number;
}

export const DEFAULT_TIMEOUT_MS = 10_000;

// HTTP/2 responses carry no reason phrase, so the common ones are named here.
const reasonPhrases: Record<number, string> = {
  200: "OK",
  400: "Bad Request",
  404: "Not Found",
  405: "Method Not Allowed",
  429: "Too Many Requests",
  500: "Internal Server Error",
  503: "Service Unavailable",
};

function prettyPrint(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}

/**
 * Sends a GET request and describes the outcome. Error statuses (4xx, 5xx)
 * are API responses like any other; only a failed connection or a timeout
 * is reported as an error.
 */
export async function sendRequest(
  url: string,
  {
    fetch: fetchFn = fetch,
    now = () => performance.now(),
    timeoutMs = DEFAULT_TIMEOUT_MS,
  }: SendOptions = {},
): Promise<ConsoleResult> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  const start = now();
  const elapsed = () => Math.round(now() - start);

  try {
    const response = await fetchFn(url, {
      headers: { Accept: "application/json" },
      signal: controller.signal,
    });
    const text = await response.text();
    return {
      kind: "response",
      status: response.status,
      statusText: response.statusText || reasonPhrases[response.status] || "",
      ok: response.ok,
      durationMs: elapsed(),
      sizeBytes: new TextEncoder().encode(text).length,
      requestId: response.headers.get("X-Request-ID"),
      body: prettyPrint(text),
    };
  } catch {
    return controller.signal.aborted
      ? { kind: "timeout", durationMs: elapsed() }
      : { kind: "network-error", durationMs: elapsed() };
  } finally {
    clearTimeout(timer);
  }
}
