import { vi } from "vitest";

/**
 * Mock all auth-related dependencies to prevent initialization side effects
 * Use this in tests that need to import WorkOSAuthProvider or other auth modules
 */
export const setupAuthTestMocks = () => {
  // Mock get-auth module to prevent server initialization
  vi.mock("../get-auth", () => ({
    getAuth: vi.fn().mockResolvedValue({ userId: "test-user-id" }),
  }));

  // Mock utils module
  vi.mock("../../utils", () => ({
    getBaseUrl: vi.fn().mockReturnValue("http://localhost:3000"),
  }));

  // Mock cookie modules
  vi.mock("../cookies", () => ({
    getCookie: vi.fn(),
    setCookie: vi.fn(),
    deleteCookie: vi.fn(),
    getCookieOptionsAsString: vi.fn(),
    setSessionCookie: vi.fn(),
  }));

  // Mock cookie security
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
};

/**
 * Setup mock for global fetch API
 * Returns the mock function for further customization in tests
 */
export const setupFetchMock = () => {
  const fetchMock = vi.fn();
  global.fetch = fetchMock as any;
  return fetchMock;
};

/**
 * Create a mock fetch response
 */
export const createMockFetchResponse = (data: any, ok = true, status = 200) => ({
  ok,
  status,
  json: async () => data,
  text: async () => JSON.stringify(data),
  headers: new Headers(),
});

/**
 * Mock Radar API response with specific action
 */
export const mockRadarResponse = (
  action: "allow" | "block" | "challenge",
  reason?: string,
) => {
  const response = createMockFetchResponse({
    action,
    ...(reason && { reason }),
  });

  (global.fetch as any).mockResolvedValueOnce(response);
  return response;
};

/**
 * Mock Radar API failure
 */
export const mockRadarFailure = (status = 500) => {
  const response = createMockFetchResponse({}, false, status);
  (global.fetch as any).mockResolvedValueOnce(response);
  return response;
};

/**
 * Mock Radar API network error
 */
export const mockRadarNetworkError = (errorMessage = "Network error") => {
  (global.fetch as any).mockRejectedValueOnce(new Error(errorMessage));
};
