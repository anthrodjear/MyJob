/**
 * Tests for API client (lib/api/client.ts).
 *
 * Covers: ApiError, apiFetch, apiGet, apiPost, apiPut, apiPatch, apiDelete.
 * Mocks: global fetch, localStorage.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiError, apiFetch, apiGet, apiPost, apiPut, apiPatch, apiDelete, setAuthToken } from "../client";

// Mock fetch globally
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

describe("ApiError", () => {
  it("stores status, code, message, and rawBody", () => {
    const err = new ApiError(404, "NOT_FOUND", "Job not found", "<html>404</html>");
    expect(err.status).toBe(404);
    expect(err.code).toBe("NOT_FOUND");
    expect(err.message).toBe("Job not found");
    expect(err.rawBody).toBe("<html>404</html>");
    expect(err.name).toBe("ApiError");
  });

  it("defaults rawBody to undefined", () => {
    const err = new ApiError(500, "INTERNAL", "Server error");
    expect(err.rawBody).toBeUndefined();
  });

  it("is an instance of Error", () => {
    const err = new ApiError(400, "BAD", "Bad request");
    expect(err).toBeInstanceOf(Error);
  });
});

describe("setAuthToken", () => {
  it("uses custom provider when set", async () => {
    setAuthToken(() => "my-token");
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiGet("test");

    const [, options] = mockFetch.mock.calls[0];
    expect(options.headers.get("Authorization")).toBe("Bearer my-token");

    // Cleanup
    setAuthToken(null);
  });
});

describe("apiFetch", () => {
  beforeEach(() => {
    mockFetch.mockReset();
    setAuthToken(null);
  });

  afterEach(() => {
    setAuthToken(null);
  });

  it("constructs correct URL with API prefix", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ id: 1 }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiFetch("jobs/123");

    const [url] = mockFetch.mock.calls[0];
    expect(url.toString()).toContain("/api/v1/jobs/123");
  });

  it("sets Content-Type to application/json by default", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiFetch("test");

    const [, options] = mockFetch.mock.calls[0];
    expect(options.headers.get("Content-Type")).toBe("application/json");
  });

  it("preserves existing Content-Type header", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiFetch("test", {
      headers: { "Content-Type": "multipart/form-data" },
    });

    const [, options] = mockFetch.mock.calls[0];
    expect(options.headers.get("Content-Type")).toBe("multipart/form-data");
  });

  it("returns parsed JSON on success", async () => {
    const data = { jobs: [{ id: 1 }], total: 1 };
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify(data), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const result = await apiGet<typeof data>("jobs");
    expect(result).toEqual(data);
  });

  it("returns undefined on 204 No Content", async () => {
    mockFetch.mockResolvedValueOnce(new Response(null, { status: 204 }));

    const result = await apiDelete("jobs/123");
    expect(result).toBeUndefined();
  });

  it("throws ApiError on non-2xx response with JSON body", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(
        JSON.stringify({ error: { code: "NOT_FOUND", message: "Job not found" } }),
        { status: 404, headers: { "Content-Type": "application/json" } },
      ),
    );

    try {
      await apiGet("jobs/999");
      expect.fail("Should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect(err).toMatchObject({
        status: 404,
        code: "NOT_FOUND",
        message: "Job not found",
      });
    }
  });

  it("throws ApiError on non-JSON error response", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response("<html>Bad Gateway</html>", {
        status: 502,
        headers: { "Content-Type": "text/html" },
      }),
    );

    try {
      await apiGet("test");
      expect.fail("Should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect(err).toMatchObject({
        status: 502,
        code: "UNKNOWN_ERROR",
      });
    }
  });

  it("sets cache: no-store on all requests", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiFetch("test");

    const [, options] = mockFetch.mock.calls[0];
    expect(options.cache).toBe("no-store");
  });
});

describe("apiGet", () => {
  beforeEach(() => mockFetch.mockReset());

  it("sends GET request", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiGet("jobs");

    const [, options] = mockFetch.mock.calls[0];
    expect(options.method).toBe("GET");
  });
});

describe("apiPost", () => {
  beforeEach(() => mockFetch.mockReset());

  it("sends POST request with JSON body", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ id: "new" }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiPost("jobs", { title: "Engineer" });

    const [, options] = mockFetch.mock.calls[0];
    expect(options.method).toBe("POST");
    expect(options.body).toBe(JSON.stringify({ title: "Engineer" }));
  });

  it("sends POST without body when data is undefined", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiPost("jobs");

    const [, options] = mockFetch.mock.calls[0];
    expect(options.body).toBeUndefined();
  });
});

describe("apiPut", () => {
  beforeEach(() => mockFetch.mockReset());

  it("sends PUT request with JSON body", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiPut("applications/123/status", { status: "applied" });

    const [, options] = mockFetch.mock.calls[0];
    expect(options.method).toBe("PUT");
    expect(options.body).toBe(JSON.stringify({ status: "applied" }));
  });
});

describe("apiPatch", () => {
  beforeEach(() => mockFetch.mockReset());

  it("sends PATCH request with JSON body", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await apiPatch("profile", { name: "Updated" });

    const [, options] = mockFetch.mock.calls[0];
    expect(options.method).toBe("PATCH");
    expect(options.body).toBe(JSON.stringify({ name: "Updated" }));
  });
});

describe("apiDelete", () => {
  beforeEach(() => mockFetch.mockReset());

  it("sends DELETE request", async () => {
    mockFetch.mockResolvedValueOnce(new Response(null, { status: 204 }));

    await apiDelete("jobs/123");

    const [, options] = mockFetch.mock.calls[0];
    expect(options.method).toBe("DELETE");
  });
});
