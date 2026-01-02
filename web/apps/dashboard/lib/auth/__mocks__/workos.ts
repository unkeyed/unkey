/**
 * Mock WorkOS SDK instance factory
 * Import this in your test file and use with vi.fn()
 *
 * Example:
 * ```ts
 * import { vi } from 'vitest';
 * import { createMockWorkOSInstance } from '../__mocks__/workos';
 *
 * vi.mock('@workos-inc/node', () => ({
 *   WorkOS: vi.fn().mockImplementation(() => createMockWorkOSInstance(vi)),
 * }));
 * ```
 */
export const createMockWorkOSInstance = (viFn: typeof import("vitest").vi) => ({
  userManagement: {
    createUser: viFn.fn(),
    createMagicAuth: viFn.fn(),
    listUsers: viFn.fn(),
    getUser: viFn.fn(),
    authenticateWithMagicAuth: viFn.fn(),
    authenticateWithCode: viFn.fn(),
    authenticateWithEmailVerification: viFn.fn(),
    authenticateWithOrganizationSelection: viFn.fn(),
    loadSealedSession: viFn.fn(),
    createOrganizationMembership: viFn.fn(),
    listOrganizationMemberships: viFn.fn(),
    updateOrganizationMembership: viFn.fn(),
    deleteOrganizationMembership: viFn.fn(),
    sendInvitation: viFn.fn(),
    listInvitations: viFn.fn(),
    findInvitationByToken: viFn.fn(),
    revokeInvitation: viFn.fn(),
    acceptInvitation: viFn.fn(),
    getAuthorizationUrl: viFn.fn(),
  },
  organizations: {
    createOrganization: viFn.fn(),
    getOrganization: viFn.fn(),
    updateOrganization: viFn.fn(),
  },
});
