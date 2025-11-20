# Auth Test Mocks

Reusable mocks for testing authentication functionality.

## Directory Structure

```
__mocks__/
├── env.ts       # Environment configuration mocks
├── workos.ts    # WorkOS SDK mocks
├── setup.ts     # Test setup utilities and helpers
└── README.md    # This file
```

## Usage

### Basic Setup

For most auth tests, you'll need to set up all the standard mocks at the top level:

```typescript
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

// Mock the WorkOS SDK
vi.mock("@workos-inc/node", () => ({
  WorkOS: vi.fn().mockImplementation((apiKey: string) => createMockWorkOSInstance(apiKey, vi)),
}));

// Mock fetch globally
global.fetch = vi.fn();

// Now import after mocks are set up
import { WorkOSAuthProvider } from "../workos";

describe("Your test suite", () => {
  // Your tests here
});
```

**Note:** All `vi.mock()` calls must be at the top level of your test file, before any imports that might use them. This is a Vitest requirement for proper mock hoisting.

### Environment Mocks

```typescript
import { vi } from "vitest";
import { mockWorkOSEnv, mockLocalEnv } from "../__mocks__/env";

// Use WorkOS environment
vi.mock("@/lib/env", () => ({
  env: vi.fn(() => mockWorkOSEnv()),
}));

// Or use local auth environment
vi.mock("@/lib/env", () => ({
  env: vi.fn(() => mockLocalEnv()),
}));

// Or customize the environment
vi.mock("@/lib/env", () => ({
  env: vi.fn(() => ({
    ...mockWorkOSEnv(),
    CUSTOM_VAR: "custom-value",
  })),
}));
```

### WorkOS SDK Mocks

```typescript
import { vi } from "vitest";
import { createMockWorkOSInstance } from "../__mocks__/workos";

// Standard setup
vi.mock("@workos-inc/node", () => ({
  WorkOS: vi.fn().mockImplementation((apiKey: string) =>
    createMockWorkOSInstance(apiKey, vi)
  ),
}));

// In your test, you can customize the mock instance behavior
it("should create user", () => {
  const mockWorkOS = createMockWorkOSInstance("custom-api-key", vi);
  mockWorkOS.userManagement.createUser.mockResolvedValue({
    id: "user_123",
    email: "test@example.com"
  });
  // Use mockWorkOS in your test
});
```

### Radar API Mocks

```typescript
import { mockRadarResponse, mockRadarFailure, mockRadarNetworkError } from "../__mocks__/setup";

// Mock a successful "allow" response
mockRadarResponse("allow");

// Mock a "block" response with reason
mockRadarResponse("block", "Suspicious activity detected");

// Mock an API failure
mockRadarFailure(500);

// Mock a network error
mockRadarNetworkError("Connection timeout");
```

### Fetch Mocks

```typescript
import { setupFetchMock, createMockFetchResponse } from "../__mocks__/setup";

const fetchMock = setupFetchMock();

// Mock a custom response
fetchMock.mockResolvedValueOnce(
  createMockFetchResponse({ data: "value" }, true, 200)
);
```

## Examples

### Testing Radar Integration

```typescript
import { mockRadarResponse } from "../__mocks__/setup";

it("should block signup when Radar returns block action", async () => {
  mockRadarResponse("block", "Suspicious activity detected");

  const result = await provider.signUpViaEmail({
    email: "test@example.com",
    firstName: "Test",
    lastName: "User",
  });

  expect(result.success).toBe(false);
});
```

### Testing with Custom WorkOS Responses

```typescript
it("should create user successfully", async () => {
  mockRadarResponse("allow");

  const mockProvider = {
    userManagement: {
      createUser: vi.fn().mockResolvedValue({ id: "user_123" }),
      createMagicAuth: vi.fn().mockResolvedValue({}),
    },
    key: "test-api-key",
  };

  (provider as any).provider = mockProvider;

  const result = await provider.signUpViaEmail({
    email: "test@example.com",
    firstName: "Test",
    lastName: "User",
  });

  expect(result.success).toBe(true);
  expect(mockProvider.userManagement.createUser).toHaveBeenCalled();
});
```

## Adding New Mocks

When adding new reusable mocks:

1. **env.ts** - Add new environment configurations
2. **workos.ts** - Extend WorkOS SDK mock methods
3. **setup.ts** - Add new helper utilities for common test scenarios

Keep mocks focused and composable for maximum reusability.
