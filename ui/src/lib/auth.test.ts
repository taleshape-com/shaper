import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  extractAndClearTokenFromUrl,
  getVariablesString,
  refreshJwt,
  localStorageVariablesKey,
  localStorageJwtKey,
} from "./auth";
import { loadSystemConfig, localStorageSystemConfigKey } from "./system";

describe("extractAndClearTokenFromUrl", () => {
  it("should return null and leave URL untouched when no token is present", () => {
    const res = extractAndClearTokenFromUrl("", "", "/dashboards/1");
    expect(res.token).toBeNull();
    expect(res.cleanUrl).toBe("/dashboards/1");
  });

  it("should preserve search and hash when no token is present", () => {
    const res = extractAndClearTokenFromUrl("?foo=bar", "#section-1", "/dashboards/1");
    expect(res.token).toBeNull();
    expect(res.cleanUrl).toBe("/dashboards/1?foo=bar#section-1");
  });

  it("should extract token from URL search query (?token=...) and clean it", () => {
    const res = extractAndClearTokenFromUrl("?token=my-secret-jwt", "", "/dashboards/1");
    expect(res.token).toBe("my-secret-jwt");
    expect(res.cleanUrl).toBe("/dashboards/1");
  });

  it("should preserve other search parameters when cleaning token from search", () => {
    const res = extractAndClearTokenFromUrl("?preview=true&token=my-jwt&user=admin", "", "/dashboards/1");
    expect(res.token).toBe("my-jwt");
    expect(res.cleanUrl).toBe("/dashboards/1?preview=true&user=admin");
  });

  it("should extract token from URL hash fragment (#token=...) and clean it", () => {
    const res = extractAndClearTokenFromUrl("", "#token=my-hash-jwt", "/dashboards/1");
    expect(res.token).toBe("my-hash-jwt");
    expect(res.cleanUrl).toBe("/dashboards/1");
  });

  it("should extract token from hash with leading question mark (#?token=...)", () => {
    const res = extractAndClearTokenFromUrl("", "#?token=my-hash-jwt", "/dashboards/1");
    expect(res.token).toBe("my-hash-jwt");
    expect(res.cleanUrl).toBe("/dashboards/1");
  });

  it("should preserve other hash parameters when cleaning token from hash", () => {
    const res = extractAndClearTokenFromUrl("", "#token=my-hash-jwt&theme=dark", "/dashboards/1");
    expect(res.token).toBe("my-hash-jwt");
    expect(res.cleanUrl).toBe("/dashboards/1#theme=dark");
  });

  it("should extract raw compact JWT directly from hash fragment (#ey...)", () => {
    const sampleJwt = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiIxMjMifQ.signature";
    const res = extractAndClearTokenFromUrl("", `#${sampleJwt}`, "/dashboards/1");
    expect(res.token).toBe(sampleJwt);
    expect(res.cleanUrl).toBe("/dashboards/1");
  });

  it("should extract token from hash while preserving search query parameters", () => {
    const res = extractAndClearTokenFromUrl("?preview=true", "#token=hash-token", "/dashboards/1");
    expect(res.token).toBe("hash-token");
    expect(res.cleanUrl).toBe("/dashboards/1?preview=true");
  });

  it("should prefer hash token over search token when both are present and clean both", () => {
    const res = extractAndClearTokenFromUrl("?token=search-token&foo=bar", "#token=hash-token", "/dashboards/1");
    expect(res.token).toBe("hash-token");
    expect(res.cleanUrl).toBe("/dashboards/1?foo=bar");
  });
});

describe("getVariablesString", () => {
  const store: Record<string, string> = {};

  beforeEach(() => {
    for (const k in store) delete store[k];
    const mockLocalStorage = {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, val: string) => { store[key] = val; },
      removeItem: (key: string) => { delete store[key]; },
      clear: () => { for (const k in store) delete store[k]; },
      key: () => null,
      length: 0,
    } as unknown as Storage;
    globalThis.window = {
      localStorage: mockLocalStorage,
      atob: (str: string) => Buffer.from(str, "base64").toString("binary"),
      shaper: { defaultBaseUrl: "http://localhost:5454/" },
    } as unknown as Window & typeof globalThis;
    globalThis.localStorage = mockLocalStorage;
  });

  it("should return stored variables if present", () => {
    store[localStorageVariablesKey] = JSON.stringify({ city: "Berlin" });
    expect(getVariablesString()).toBe(JSON.stringify({ city: "Berlin" }));
  });

  it("should fallback to JWT variables claim if stored variables are null", () => {
    const payload = Buffer.from(JSON.stringify({ variables: { env: "prod" } })).toString("base64");
    store[localStorageJwtKey] = `header.${payload}.sig`;
    expect(getVariablesString()).toBe(JSON.stringify({ env: "prod" }));
  });

  it("should return '{}' if stored variables are null and JWT has no variables", () => {
    const payload = Buffer.from(JSON.stringify({ userId: "123" })).toString("base64");
    store[localStorageJwtKey] = `header.${payload}.sig`;
    expect(getVariablesString()).toBe("{}");
  });

  it("should return '{}' if neither stored variables nor JWT are present", () => {
    expect(getVariablesString()).toBe("{}");
  });
});

describe("refreshJwt variables serialization", () => {
  const store: Record<string, string> = {};

  beforeEach(async () => {
    for (const k in store) delete store[k];
    const mockLocalStorage = {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, val: string) => { store[key] = val; },
      removeItem: (key: string) => { delete store[key]; },
      clear: () => { for (const k in store) delete store[k]; },
      key: () => null,
      length: 0,
    } as unknown as Storage;
    globalThis.window = {
      localStorage: mockLocalStorage,
      atob: (str: string) => Buffer.from(str, "base64").toString("binary"),
      shaper: { defaultBaseUrl: "http://localhost:5454/" },
    } as unknown as Window & typeof globalThis;
    globalThis.localStorage = mockLocalStorage;
    store[localStorageSystemConfigKey] = JSON.stringify({
      loginRequired: false,
      tasksEnabled: false,
      editEnabled: true,
      publicSharingEnabled: true,
      passwordProtectedSharingEnabled: true,
    });
    await loadSystemConfig();
  });

  it("should send empty variables object in body when vars is empty", async () => {
    let capturedBody: any = null;
    globalThis.fetch = vi.fn().mockImplementation(async (url: string, options?: any) => {
      if (typeof url === "string" && url.includes("api/system/config")) {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            loginRequired: false,
            tasksEnabled: false,
            editEnabled: true,
            publicSharingEnabled: true,
            passwordProtectedSharingEnabled: true,
          }),
        };
      }
      if (options?.body) {
        capturedBody = JSON.parse(options.body);
      }
      return {
        ok: true,
        status: 200,
        json: async () => ({ jwt: "new-jwt-token" }),
      };
    });

    const res = await refreshJwt("token-123", {});
    expect(res).toBe("new-jwt-token");
    expect(capturedBody).toBeDefined();
    expect(capturedBody.variables).toEqual({});
  });

  it("should send non-empty variables object in body when vars has values", async () => {
    let capturedBody: any = null;
    globalThis.fetch = vi.fn().mockImplementation(async (url: string, options?: any) => {
      if (typeof url === "string" && url.includes("api/system/config")) {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            loginRequired: false,
            tasksEnabled: false,
            editEnabled: true,
            publicSharingEnabled: true,
            passwordProtectedSharingEnabled: true,
          }),
        };
      }
      if (options?.body) {
        capturedBody = JSON.parse(options.body);
      }
      return {
        ok: true,
        status: 200,
        json: async () => ({ jwt: "new-jwt-token" }),
      };
    });

    const res = await refreshJwt("token-123", { city: "Berlin" });
    expect(res).toBe("new-jwt-token");
    expect(capturedBody).toBeDefined();
    expect(capturedBody.variables).toEqual({ city: "Berlin" });
  });
});
