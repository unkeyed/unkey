import { beforeEach, describe, expect, it, vi } from "vitest";
import { mockWorkOSEnv } from "../__mocks__/env";
import {
  mockRadarFailure,
  mockRadarNetworkError,
  mockRadarResponse,
  setupFetchMock,
} from "../__mocks__/setup";
import { createMockWorkOSInstance } from "../__mocks__/workos";

// Mock the env module BEFORE importing anything else
vi.mock("@/lib/env", () => ({
  env: vi.fn(() => mockWorkOSEnv()),
}));

// Mock the get-auth module to prevent server initialization
vi.mock("../get-auth", () => ({
  getAuth: vi.fn().mockResolvedValue({ userId: "test-user-id" }),
}));

// Mock the utils module
vi.mock("../../utils", () => ({
  getBaseUrl: vi.fn().mockReturnValue("http://localhost:3000"),
}));

// Mock the cookie modules
vi.mock("../cookies", () => ({
  getCookie: vi.fn(),
  setCookie: vi.fn(),
  deleteCookie: vi.fn(),
  getCookieOptionsAsString: vi.fn(),
  setSessionCookie: vi.fn(),
}));

vi.mock("../cookie-security", () => ({
  getAuthCookieOptions: vi.fn().mockReturnValue({
    httpOnly: true,
    secure: false,
    sameSite: "lax",
    path: "/",
  }),
  getDefaultCookieOptions: vi.fn(),
  shouldUseSecureCookies: vi.fn(),
}));

// Mock the WorkOS SDK
vi.mock("@workos-inc/node", () => ({
  WorkOS: vi.fn().mockImplementation((apiKey: string) => createMockWorkOSInstance(vi, apiKey)),
}));

// Now import after mocks are set up
import { WorkOSAuthProvider } from "../workos";

describe("WorkOSAuthProvider - checkRadar", () => {
  let provider: WorkOSAuthProvider;

  beforeEach(() => {
    vi.clearAllMocks();
    setupFetchMock();
    provider = new WorkOSAuthProvider({
      apiKey: "test-api-key",
      clientId: "test-client-id",
    });
  });

  describe("signUpViaEmail with Radar checks", () => {
    it("should block signup when Radar returns block action", async () => {
      mockRadarResponse("block", "Suspicious activity detected");

      const result = await provider.signUpViaEmail({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(result).toEqual({
        success: false,
        code: "UNKNOWN_ERROR",
        message: "Suspicious activity detected",
      });

      // Verify Radar API was called with correct parameters
      expect(global.fetch).toHaveBeenCalledWith(
        "https://api.workos.com/radar/events",
        expect.objectContaining({
          method: "POST",
          headers: {
            Authorization: "Bearer test-api-key",
            "Content-Type": "application/json",
          },
          body: expect.stringContaining("test@example.com"),
        }),
      );
    });

    it("should allow signup when Radar returns allow action", async () => {
      mockRadarResponse("allow");

      // Mock WorkOS user creation and magic auth
      const mockProvider = {
        userManagement: {
          createUser: vi.fn().mockResolvedValue({}),
          createMagicAuth: vi.fn().mockResolvedValue({}),
        },
        key: "test-api-key",
      };

      // Replace the provider instance
      (provider as any).provider = mockProvider;

      const result = await provider.signUpViaEmail({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(result).toEqual({ success: true });
      expect(mockProvider.userManagement.createUser).toHaveBeenCalledWith({
        firstName: "Test",
        lastName: "User",
        email: "test@example.com",
      });
      expect(mockProvider.userManagement.createMagicAuth).toHaveBeenCalledWith({
        email: "test@example.com",
      });
    });

    it("should allow signup when Radar returns challenge action", async () => {
      mockRadarResponse("challenge", "Additional verification recommended");

      // Mock WorkOS user creation and magic auth
      const mockProvider = {
        userManagement: {
          createUser: vi.fn().mockResolvedValue({}),
          createMagicAuth: vi.fn().mockResolvedValue({}),
        },
        key: "test-api-key",
      };

      (provider as any).provider = mockProvider;

      const result = await provider.signUpViaEmail({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(result).toEqual({ success: true });
      expect(mockProvider.userManagement.createUser).toHaveBeenCalled();
    });

    it("should allow signup when Radar API fails", async () => {
      mockRadarFailure(500);

      // Mock WorkOS user creation and magic auth
      const mockProvider = {
        userManagement: {
          createUser: vi.fn().mockResolvedValue({}),
          createMagicAuth: vi.fn().mockResolvedValue({}),
        },
        key: "test-api-key",
      };

      (provider as any).provider = mockProvider;

      const result = await provider.signUpViaEmail({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(result).toEqual({ success: true });
      expect(mockProvider.userManagement.createUser).toHaveBeenCalled();
    });

    it("should allow signup when Radar API throws an error", async () => {
      mockRadarNetworkError("Network error");

      // Mock WorkOS user creation and magic auth
      const mockProvider = {
        userManagement: {
          createUser: vi.fn().mockResolvedValue({}),
          createMagicAuth: vi.fn().mockResolvedValue({}),
        },
        key: "test-api-key",
      };

      (provider as any).provider = mockProvider;

      const result = await provider.signUpViaEmail({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(result).toEqual({ success: true });
      expect(mockProvider.userManagement.createUser).toHaveBeenCalled();
    });
  });

  describe("signInViaEmail with Radar checks", () => {
    it("should block signin when Radar returns block action", async () => {
      mockRadarResponse("block", "Account compromised");

      const result = await provider.signInViaEmail({
        email: "test@example.com",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(result).toEqual({
        success: false,
        code: "UNKNOWN_ERROR",
        message: "Account compromised",
      });
    });

    it("should allow signin when Radar returns allow action", async () => {
      mockRadarResponse("allow");

      // Mock WorkOS listUsers and createMagicAuth
      const mockProvider = {
        userManagement: {
          listUsers: vi.fn().mockResolvedValue({
            data: [{ id: "user_123", email: "test@example.com" }],
          }),
          createMagicAuth: vi.fn().mockResolvedValue({}),
        },
        key: "test-api-key",
      };

      (provider as any).provider = mockProvider;

      const result = await provider.signInViaEmail({
        email: "test@example.com",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(result).toEqual({ success: true });
      expect(mockProvider.userManagement.listUsers).toHaveBeenCalledWith({
        email: "test@example.com",
      });
    });

    it("should allow signin when Radar returns challenge action", async () => {
      mockRadarResponse("challenge", "Unusual location detected");

      const mockProvider = {
        userManagement: {
          listUsers: vi.fn().mockResolvedValue({
            data: [{ id: "user_123", email: "test@example.com" }],
          }),
          createMagicAuth: vi.fn().mockResolvedValue({}),
        },
        key: "test-api-key",
      };

      (provider as any).provider = mockProvider;

      const result = await provider.signInViaEmail({
        email: "test@example.com",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(result).toEqual({ success: true });
      expect(mockProvider.userManagement.listUsers).toHaveBeenCalled();
    });
  });
});
