import { describe, expect, it, vi } from "vitest";
import {
  buildUrl,
  consoleEndpoints,
  sendRequest,
  splitPath,
} from "../api-console";

/** A clock that advances `step` milliseconds every time it is read. */
function fakeClock(step: number) {
  let time = 0;
  return () => (time += step) - step;
}

describe("splitPath", () => {
  it.each([
    {
      path: "/v1/states/{code}",
      want: { prefix: "/v1/states/", param: "code", suffix: "" },
    },
    {
      path: "/v1/holidays/national?year={year}",
      want: {
        prefix: "/v1/holidays/national?year=",
        param: "year",
        suffix: "",
      },
    },
    {
      path: "/v1/states/{code}/phone-prefixes",
      want: { prefix: "/v1/states/", param: "code", suffix: "/phone-prefixes" },
    },
    {
      path: "/health",
      want: { prefix: "/health", param: "", suffix: "" },
    },
  ])("splits $path", ({ path, want }) => {
    expect(splitPath(path)).toEqual(want);
  });

  it("finds a parameter in every console endpoint", () => {
    for (const endpoint of consoleEndpoints) {
      expect(splitPath(endpoint.path).param).not.toBe("");
    }
  });
});

describe("buildUrl", () => {
  it.each([
    {
      name: "joins base and path",
      base: "http://localhost:8080",
      value: "VE14",
      want: "http://localhost:8080/v1/states/VE14",
    },
    {
      name: "does not duplicate slashes",
      base: "https://api.example.org/",
      value: "VE14",
      want: "https://api.example.org/v1/states/VE14",
    },
    {
      name: "trims and encodes the value",
      base: "http://localhost:8080",
      value: "  san cristóbal/x ",
      want: "http://localhost:8080/v1/states/san%20crist%C3%B3bal%2Fx",
    },
  ])("$name", ({ base, value, want }) => {
    expect(buildUrl(base, "/v1/states/{code}", value)).toBe(want);
  });
});

describe("sendRequest", () => {
  it("describes a successful response", async () => {
    const body = '{"data":{"code":"0102"}}';
    const fetch = vi.fn().mockResolvedValue(
      new Response(body, {
        status: 200,
        headers: { "X-Request-ID": "abc123" },
      }),
    );

    const result = await sendRequest("http://api/v1/banks/0102", {
      fetch,
      now: fakeClock(7),
    });

    expect(result).toEqual({
      kind: "response",
      status: 200,
      statusText: "OK",
      ok: true,
      durationMs: 7,
      sizeBytes: body.length,
      requestId: "abc123",
      body: '{\n  "data": {\n    "code": "0102"\n  }\n}',
    });
    expect(fetch).toHaveBeenCalledWith(
      "http://api/v1/banks/0102",
      expect.objectContaining({ headers: { Accept: "application/json" } }),
    );
  });

  it("treats an error status as a response", async () => {
    const fetch = vi
      .fn()
      .mockResolvedValue(new Response('{"status":404}', { status: 404 }));

    const result = await sendRequest("http://api/v1/banks/9999", { fetch });

    expect(result).toMatchObject({
      kind: "response",
      status: 404,
      statusText: "Not Found",
      ok: false,
      requestId: null,
    });
  });

  it("counts the size in bytes, not characters", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response('"ñ"'));
    const result = await sendRequest("http://api/x", { fetch });
    expect(result).toMatchObject({ sizeBytes: 4 });
  });

  it("keeps a body that is not JSON as it is", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response("Bad Gateway"));
    const result = await sendRequest("http://api/x", { fetch });
    expect(result).toMatchObject({ body: "Bad Gateway" });
  });

  it("reports a failed connection", async () => {
    const fetch = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));
    const result = await sendRequest("http://api/x", {
      fetch,
      now: fakeClock(3),
    });
    expect(result).toEqual({ kind: "network-error", durationMs: 3 });
  });

  it("gives up after the timeout", async () => {
    const fetch = vi.fn(
      (_url: string | URL | Request, init?: RequestInit) =>
        new Promise<Response>((_, reject) => {
          init?.signal?.addEventListener("abort", () =>
            reject(new DOMException("Aborted", "AbortError")),
          );
        }),
    );

    const result = await sendRequest("http://api/x", {
      fetch: fetch as typeof globalThis.fetch,
      timeoutMs: 5,
    });

    expect(result).toMatchObject({ kind: "timeout" });
  });
});
