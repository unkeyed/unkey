import { beforeEach, describe, expect, it, vi } from "vitest";
import { mockWorkOSEnv } from "../__mocks__/env";
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

// Mock the WorkOS SDK but keep the real exception classes so the provider's
// instanceof checks work
vi.mock("@workos-inc/node", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@workos-inc/node")>();
  return {
    ...actual,
    WorkOS: vi.fn().mockImplementation(() => createMockWorkOSInstance(vi)),
  };
});

// Now import after mocks are set up
import { AuthenticationException, WorkOS } from "@workos-inc/node";
import {
  AUTH_CHALLENGE_COOKIE,
  AuthErrorCode,
  PENDING_SESSION_COOKIE,
  RADAR_ATTEMPT_COOKIE,
  UNKEY_SESSION_COOKIE,
} from "../types";
import { WorkOSAuthProvider } from "../workos";

type MockWorkOS = ReturnType<typeof createMockWorkOSInstance>;

function getMockInstance(): MockWorkOS {
  const results = vi.mocked(WorkOS).mock.results;
  return results[results.length - 1].value as MockWorkOS;
}

function authException(
  rawData: Record<string, unknown> & { code: string },
): AuthenticationException {
  type RawData = ConstructorParameters<typeof AuthenticationException>[1];
  return new AuthenticationException(403, rawData as unknown as RawData, "req_test");
}

describe("WorkOSAuthProvider", () => {
  let provider: WorkOSAuthProvider;
  let workos: MockWorkOS;

  beforeEach(() => {
    vi.clearAllMocks();
    provider = new WorkOSAuthProvider({
      apiKey: "test-api-key",
      clientId: "test-client-id",
    });
    workos = getMockInstance();
  });

  describe("signUpViaEmail", () => {
    it("passes request metadata to createUser and threads the Radar attempt into createMagicAuth", async () => {
      workos.userManagement.createUser.mockResolvedValue({
        id: "user_123",
        radarAuthAttemptId: "radar_attempt_1",
      });
      workos.userManagement.createMagicAuth.mockResolvedValue({
        id: "magic_auth_1",
        radarAuthAttemptId: "radar_attempt_1",
      });

      const result = await provider.signUpViaEmail({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });

      expect(workos.userManagement.createUser).toHaveBeenCalledWith({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
      });
      expect(workos.userManagement.createMagicAuth).toHaveBeenCalledWith({
        email: "test@example.com",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
        radarAuthAttemptId: "radar_attempt_1",
      });
      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.cookies).toEqual([
          expect.objectContaining({
            name: RADAR_ATTEMPT_COOKIE,
            value: "radar_attempt_1",
          }),
        ]);
      }
    });

    it("resends the code when the email belongs to an unverified user", async () => {
      workos.userManagement.createUser.mockRejectedValue({
        errors: [{ code: "email_not_available" }],
      });
      workos.userManagement.listUsers.mockResolvedValue({
        data: [{ id: "user_123", email: "test@example.com", emailVerified: false }],
      });
      workos.userManagement.createMagicAuth.mockResolvedValue({ id: "magic_auth_1" });

      const result = await provider.signUpViaEmail({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
      });

      expect(result.success).toBe(true);
      expect(workos.userManagement.createMagicAuth).toHaveBeenCalledWith({
        email: "test@example.com",
        ipAddress: undefined,
        userAgent: undefined,
      });
    });

    it("returns EMAIL_ALREADY_EXISTS when the email belongs to a verified user", async () => {
      workos.userManagement.createUser.mockRejectedValue({
        errors: [{ code: "email_not_available" }],
      });
      workos.userManagement.listUsers.mockResolvedValue({
        data: [{ id: "user_123", email: "test@example.com", emailVerified: true }],
      });

      const result = await provider.signUpViaEmail({
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.EMAIL_ALREADY_EXISTS);
      }
      expect(workos.userManagement.createMagicAuth).not.toHaveBeenCalled();
    });
  });

  describe("verifyAuthCode", () => {
    it("passes Radar metadata to authenticateWithMagicAuth and clears transient cookies on success", async () => {
      workos.userManagement.authenticateWithMagicAuth.mockResolvedValue({
        sealedSession: "sealed_123",
      });

      const result = await provider.verifyAuthCode({
        email: "test@example.com",
        code: "123456",
        ipAddress: "192.168.1.1",
        userAgent: "Mozilla/5.0",
        radarAuthAttemptId: "radar_attempt_1",
      });

      expect(workos.userManagement.authenticateWithMagicAuth).toHaveBeenCalledWith(
        expect.objectContaining({
          email: "test@example.com",
          code: "123456",
          ipAddress: "192.168.1.1",
          userAgent: "Mozilla/5.0",
          radarAuthAttemptId: "radar_attempt_1",
        }),
      );
      expect(result.success).toBe(true);
      if (result.success) {
        const names = result.cookies.map((cookie) => cookie.name);
        expect(names).toContain(UNKEY_SESSION_COOKIE);
        expect(names).toContain(AUTH_CHALLENGE_COOKIE);
        expect(names).toContain(RADAR_ATTEMPT_COOKIE);
      }
    });

    it("maps a radar_email_challenge error to a pending challenge response", async () => {
      workos.userManagement.authenticateWithMagicAuth.mockRejectedValue(
        authException({
          code: "radar_email_challenge",
          pending_authentication_token: "pending_token_1",
          radar_challenge_id: "radar_challenge_1",
        }),
      );

      const result = await provider.verifyAuthCode({
        email: "test@example.com",
        code: "123456",
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.RADAR_EMAIL_CHALLENGE_REQUIRED);
        expect("challengeType" in result && result.challengeType).toBe("radar-email");
        const cookies = "cookies" in result ? (result.cookies ?? []) : [];
        expect(cookies).toContainEqual(
          expect.objectContaining({
            name: PENDING_SESSION_COOKIE,
            value: "pending_token_1",
          }),
        );
        const challengeCookie = cookies.find((cookie) => cookie.name === AUTH_CHALLENGE_COOKIE);
        expect(challengeCookie).toBeDefined();
        expect(JSON.parse(challengeCookie?.value ?? "{}")).toEqual({
          type: "radar-email",
          radarChallengeId: "radar_challenge_1",
        });
      }
    });

    it("maps a radar_sms_challenge error to a pending challenge response", async () => {
      workos.userManagement.authenticateWithMagicAuth.mockRejectedValue(
        authException({
          code: "radar_sms_challenge",
          pending_authentication_token: "pending_token_1",
          user: { id: "user_123", email: "test@example.com" },
        }),
      );

      const result = await provider.verifyAuthCode({
        email: "test@example.com",
        code: "123456",
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.RADAR_SMS_CHALLENGE_REQUIRED);
        const cookies = "cookies" in result ? (result.cookies ?? []) : [];
        const challengeCookie = cookies.find((cookie) => cookie.name === AUTH_CHALLENGE_COOKIE);
        expect(JSON.parse(challengeCookie?.value ?? "{}")).toEqual({
          type: "radar-sms",
          userId: "user_123",
        });
      }
    });

    it("creates a TOTP challenge when an mfa_challenge error is thrown", async () => {
      workos.userManagement.authenticateWithMagicAuth.mockRejectedValue(
        authException({
          code: "mfa_challenge",
          pending_authentication_token: "pending_token_1",
          user: { id: "user_123", email: "test@example.com" },
          authentication_factors: [{ id: "auth_factor_1", type: "totp" }],
        }),
      );
      workos.multiFactorAuth.challengeFactor.mockResolvedValue({ id: "auth_challenge_1" });

      const result = await provider.verifyAuthCode({
        email: "test@example.com",
        code: "123456",
      });

      expect(workos.multiFactorAuth.challengeFactor).toHaveBeenCalledWith({
        authenticationFactorId: "auth_factor_1",
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.MFA_CHALLENGE_REQUIRED);
        const cookies = "cookies" in result ? (result.cookies ?? []) : [];
        const challengeCookie = cookies.find((cookie) => cookie.name === AUTH_CHALLENGE_COOKIE);
        expect(JSON.parse(challengeCookie?.value ?? "{}")).toEqual({
          type: "mfa",
          challengeId: "auth_challenge_1",
        });
      }
    });

    it("maps an mfa_enrollment error to a pending enrollment response", async () => {
      workos.userManagement.authenticateWithMagicAuth.mockRejectedValue(
        authException({
          code: "mfa_enrollment",
          pending_authentication_token: "pending_token_1",
          user: { id: "user_123", email: "test@example.com" },
        }),
      );

      const result = await provider.verifyAuthCode({
        email: "test@example.com",
        code: "123456",
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.MFA_ENROLLMENT_REQUIRED);
        const cookies = "cookies" in result ? (result.cookies ?? []) : [];
        const challengeCookie = cookies.find((cookie) => cookie.name === AUTH_CHALLENGE_COOKIE);
        expect(JSON.parse(challengeCookie?.value ?? "{}")).toEqual({
          type: "mfa-enroll",
          userId: "user_123",
          email: "test@example.com",
        });
      }
    });

    it("maps organization_selection_required to the org selection response", async () => {
      workos.userManagement.authenticateWithMagicAuth.mockRejectedValue(
        authException({
          code: "organization_selection_required",
          pending_authentication_token: "pending_token_1",
          user: { id: "user_123", email: "test@example.com" },
          organizations: [{ id: "org_1", name: "Org One" }],
        }),
      );

      const result = await provider.verifyAuthCode({
        email: "test@example.com",
        code: "123456",
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED);
        expect("organizations" in result && result.organizations).toEqual([
          { id: "org_1", name: "Org One" },
        ]);
      }
    });
  });

  describe("challenge completion", () => {
    it("completes an MFA challenge with authenticateWithTotp", async () => {
      workos.userManagement.authenticateWithTotp.mockResolvedValue({
        sealedSession: "sealed_123",
      });

      const result = await provider.completeMfaChallenge({
        code: "123456",
        challengeId: "auth_challenge_1",
        pendingAuthToken: "pending_token_1",
      });

      expect(workos.userManagement.authenticateWithTotp).toHaveBeenCalledWith(
        expect.objectContaining({
          code: "123456",
          authenticationChallengeId: "auth_challenge_1",
          pendingAuthenticationToken: "pending_token_1",
        }),
      );
      expect(result.success).toBe(true);
    });

    it("completes a Radar email challenge", async () => {
      workos.userManagement.authenticateWithRadarEmailChallenge.mockResolvedValue({
        sealedSession: "sealed_123",
      });

      const result = await provider.completeRadarEmailChallenge({
        code: "123456",
        radarChallengeId: "radar_challenge_1",
        pendingAuthToken: "pending_token_1",
      });

      expect(workos.userManagement.authenticateWithRadarEmailChallenge).toHaveBeenCalledWith(
        expect.objectContaining({
          code: "123456",
          radarChallengeId: "radar_challenge_1",
          pendingAuthenticationToken: "pending_token_1",
        }),
      );
      expect(result.success).toBe(true);
    });

    it("completes a Radar SMS challenge end to end", async () => {
      workos.userManagement.sendRadarSmsChallenge.mockResolvedValue({
        verificationId: "verification_1",
        phoneNumber: "+15555555555",
      });
      workos.userManagement.authenticateWithRadarSmsChallenge.mockResolvedValue({
        sealedSession: "sealed_123",
      });

      const sendResult = await provider.sendRadarSmsCode({
        userId: "user_123",
        phoneNumber: "+15555555555",
        pendingAuthToken: "pending_token_1",
      });
      expect(sendResult).toEqual({
        verificationId: "verification_1",
        phoneNumber: "+15555555555",
      });

      const result = await provider.completeRadarSmsChallenge({
        code: "123456",
        verificationId: "verification_1",
        phoneNumber: "+15555555555",
        pendingAuthToken: "pending_token_1",
      });
      expect(result.success).toBe(true);
    });

    it("surfaces a follow-up org selection after completing an MFA challenge", async () => {
      workos.userManagement.authenticateWithTotp.mockRejectedValue(
        authException({
          code: "organization_selection_required",
          pending_authentication_token: "pending_token_2",
          user: { id: "user_123", email: "test@example.com" },
          organizations: [{ id: "org_1", name: "Org One" }],
        }),
      );

      const result = await provider.completeMfaChallenge({
        code: "123456",
        challengeId: "auth_challenge_1",
        pendingAuthToken: "pending_token_1",
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED);
      }
    });
  });

  describe("completeOrgSelection with org-level MFA", () => {
    it("requires MFA enrollment when the selected org enforces it and the user has no factor", async () => {
      workos.userManagement.authenticateWithOrganizationSelection.mockRejectedValue(
        authException({
          code: "mfa_enrollment",
          pending_authentication_token: "pending_token_2",
          user: { id: "user_123", email: "test@example.com" },
        }),
      );

      const result = await provider.completeOrgSelection({
        orgId: "org_1",
        pendingAuthToken: "pending_token_1",
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.MFA_ENROLLMENT_REQUIRED);
        expect("challengeType" in result && result.challengeType).toBe("mfa-enroll");
        const cookies = "cookies" in result ? (result.cookies ?? []) : [];
        const challengeCookie = cookies.find((cookie) => cookie.name === AUTH_CHALLENGE_COOKIE);
        expect(JSON.parse(challengeCookie?.value ?? "{}")).toEqual({
          type: "mfa-enroll",
          userId: "user_123",
          email: "test@example.com",
        });
        expect(cookies).toContainEqual(
          expect.objectContaining({ name: PENDING_SESSION_COOKIE, value: "pending_token_2" }),
        );
      }
    });

    it("creates a TOTP challenge when the selected org requires MFA and the user is enrolled", async () => {
      workos.userManagement.authenticateWithOrganizationSelection.mockRejectedValue(
        authException({
          code: "mfa_challenge",
          pending_authentication_token: "pending_token_2",
          user: { id: "user_123", email: "test@example.com" },
          authentication_factors: [{ id: "auth_factor_1", type: "totp" }],
        }),
      );
      workos.multiFactorAuth.challengeFactor.mockResolvedValue({ id: "auth_challenge_1" });

      const result = await provider.completeOrgSelection({
        orgId: "org_1",
        pendingAuthToken: "pending_token_1",
      });

      expect(workos.multiFactorAuth.challengeFactor).toHaveBeenCalledWith({
        authenticationFactorId: "auth_factor_1",
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.code).toBe(AuthErrorCode.MFA_CHALLENGE_REQUIRED);
        expect("challengeType" in result && result.challengeType).toBe("mfa");
      }
    });
  });

  describe("validateSession", () => {
    it("requests a refresh only for an expired access token", async () => {
      workos.userManagement.loadSealedSession.mockReturnValue({
        authenticate: vi.fn().mockResolvedValue({ authenticated: false, reason: "invalid_jwt" }),
      });

      const result = await provider.validateSession("sealed_token");
      expect(result).toEqual({ isValid: false, shouldRefresh: true });
    });

    it("skips the refresh round trip for an undecryptable cookie", async () => {
      workos.userManagement.loadSealedSession.mockReturnValue({
        authenticate: vi
          .fn()
          .mockResolvedValue({ authenticated: false, reason: "invalid_session_cookie" }),
      });

      const result = await provider.validateSession("sealed_token");
      expect(result).toEqual({ isValid: false, shouldRefresh: false });
    });

    it("falls back to a refresh attempt when validation throws transiently", async () => {
      workos.userManagement.loadSealedSession.mockReturnValue({
        authenticate: vi.fn().mockRejectedValue(new Error("fetch failed")),
      });

      const result = await provider.validateSession("sealed_token");
      expect(result).toEqual({ isValid: false, shouldRefresh: true });
    });
  });

  describe("MFA factor management", () => {
    it("begins enrollment and returns the TOTP secrets", async () => {
      workos.multiFactorAuth.createUserAuthFactor.mockResolvedValue({
        authenticationFactor: {
          id: "auth_factor_1",
          totp: { qrCode: "data:image/png;base64,abc", secret: "SECRET", uri: "otpauth://..." },
        },
        authenticationChallenge: { id: "auth_challenge_1" },
      });

      const enrollment = await provider.beginMfaEnrollment({
        userId: "user_123",
        email: "test@example.com",
      });

      expect(workos.multiFactorAuth.createUserAuthFactor).toHaveBeenCalledWith({
        userId: "user_123",
        type: "totp",
        totpIssuer: "Unkey",
        totpUser: "test@example.com",
      });
      expect(enrollment).toEqual({
        factorId: "auth_factor_1",
        challengeId: "auth_challenge_1",
        qrCode: "data:image/png;base64,abc",
        secret: "SECRET",
        uri: "otpauth://...",
      });
    });

    it("lists factors for a user", async () => {
      workos.multiFactorAuth.listUserAuthFactors.mockResolvedValue({
        data: [
          {
            id: "auth_factor_1",
            type: "totp",
            totp: { issuer: "Unkey", user: "test@example.com" },
            createdAt: "2026-01-01T00:00:00.000Z",
          },
        ],
      });

      const factors = await provider.listMfaFactors("user_123");

      expect(factors).toEqual([
        {
          id: "auth_factor_1",
          type: "totp",
          issuer: "Unkey",
          user: "test@example.com",
          createdAt: "2026-01-01T00:00:00.000Z",
        },
      ]);
    });
  });
});
