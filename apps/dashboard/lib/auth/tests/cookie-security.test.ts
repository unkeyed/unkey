import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  getAuthCookieOptions,
  getDefaultCookieOptions,
  shouldUseSecureCookies,
} from "./cookie-security";

// Mock the env module
vi.mock("@/lib/env", () => ({
  env: vi.fn(),
}));

import { env } from "@/lib/env";

const mockEnv = vi.mocked(env);

describe("cookie-security", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("shouldUseSecureCookies", () => {
    it("should return true for production environment", () => {
      mockEnv.mockReturnValue({
        VERCEL_ENV: "production",
      } as any);

      expect(shouldUseSecureCookies()).toBe(true);
    });

    it("should return false for development environment", () => {
      mockEnv.mockReturnValue({
        VERCEL_ENV: "development",
      } as any);

      expect(shouldUseSecureCookies()).toBe(false);
    });

    it("should return false for preview environment", () => {
      mockEnv.mockReturnValue({
        VERCEL_ENV: "preview",
      } as any);

      expect(shouldUseSecureCookies()).toBe(false);
    });
  });

  describe("getDefaultCookieOptions", () => {
    it("should return secure options for production", () => {
      mockEnv.mockReturnValue({
        VERCEL_ENV: "production",
      } as any);

      const options = getDefaultCookieOptions();

      expect(options).toEqual({
        httpOnly: true,
        secure: true,
        sameSite: "strict",
        path: "/",
      });
    });

    it("should return non-secure options for development", () => {
      mockEnv.mockReturnValue({
        VERCEL_ENV: "development",
      } as any);

      const options = getDefaultCookieOptions();

      expect(options).toEqual({
        httpOnly: true,
        secure: false,
        sameSite: "strict",
        path: "/",
      });
    });
  });

  describe("getAuthCookieOptions", () => {
    it("should return secure options with lax sameSite for production", () => {
      mockEnv.mockReturnValue({
        VERCEL_ENV: "production",
      } as any);

      const options = getAuthCookieOptions();

      expect(options).toEqual({
        httpOnly: true,
        secure: true,
        sameSite: "lax",
        path: "/",
      });
    });

    it("should return non-secure options with lax sameSite for development", () => {
      mockEnv.mockReturnValue({
        VERCEL_ENV: "development",
      } as any);

      const options = getAuthCookieOptions();

      expect(options).toEqual({
        httpOnly: true,
        secure: false,
        sameSite: "lax",
        path: "/",
      });
    });
  });
});
