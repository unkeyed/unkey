/**
 * Mock environment configuration for WorkOS auth tests
 * Use this in vi.mock() calls at the top of your test files
 */
export const mockWorkOSEnv = () => ({
  AUTH_PROVIDER: "workos" as const,
  WORKOS_COOKIE_PASSWORD: "test-cookie-password-32-chars-long!!",
  WORKOS_API_KEY: "test-api-key",
  WORKOS_CLIENT_ID: "test-client-id",
});

/**
 * Mock environment configuration for local auth tests
 * Use this in vi.mock() calls at the top of your test files
 */
export const mockLocalEnv = () => ({
  AUTH_PROVIDER: "local" as const,
});
