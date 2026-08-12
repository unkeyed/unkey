import { BaseAuthProvider } from "./base-provider";
import { mapWorkOSUser } from "./map-workos-user";
import {
  type MembershipListResponse,
  type Organization,
  OrganizationScopeError,
  type UpdateOrgParams,
  type User,
} from "./types";

type ProviderOrganization = {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
};

/**
 * WorkOS administrative API adapter.
 *
 * Production authentication, sessions, organization selection, and MFA are
 * handled by AuthKit. Managed widgets own interactive team administration;
 * this adapter only powers server-side organization and membership consumers.
 */
export class WorkOSAuthProvider extends BaseAuthProvider {
  private async getProvider() {
    const { getWorkOS } = await import("@workos-inc/authkit-nextjs");
    return getWorkOS();
  }

  async getUser(userId: string): Promise<User | null> {
    if (!userId) {
      throw new Error("User Id is required.");
    }
    try {
      const provider = await this.getProvider();
      return mapWorkOSUser(await provider.userManagement.getUser(userId));
    } catch {
      return null;
    }
  }

  async createTenant(params: { name: string; userId: string }): Promise<string> {
    if (!params.name || !params.userId) {
      throw new Error("Organization name and userId are required.");
    }
    try {
      const provider = await this.getProvider();
      const organization = await this.createOrg(params.name);
      const membership = await provider.userManagement.createOrganizationMembership({
        organizationId: organization.id,
        userId: params.userId,
        roleSlug: "admin",
      });
      return membership.organizationId;
    } catch (error) {
      throw this.providerError(error);
    }
  }

  protected async createOrg(name: string): Promise<Organization> {
    if (!name) {
      throw new Error("Organization name is required.");
    }
    try {
      const provider = await this.getProvider();
      return this.transformOrganization(await provider.organizations.createOrganization({ name }));
    } catch (error) {
      throw this.providerError(error);
    }
  }

  async getOrg(orgId: string): Promise<Organization> {
    if (!orgId) {
      throw new Error("Organization Id is required.");
    }
    try {
      const provider = await this.getProvider();
      return this.transformOrganization(await provider.organizations.getOrganization(orgId));
    } catch (error) {
      throw this.providerError(error);
    }
  }

  async updateOrg(params: UpdateOrgParams): Promise<Organization> {
    if (!params.id || !params.name) {
      throw new Error("Organization id and name are required.");
    }
    try {
      const provider = await this.getProvider();
      return this.transformOrganization(
        await provider.organizations.updateOrganization({
          organization: params.id,
          name: params.name,
        }),
      );
    } catch (error) {
      throw this.providerError(error);
    }
  }

  async listMemberships(userId: string): Promise<MembershipListResponse> {
    try {
      const provider = await this.getProvider();
      const [user, memberships] = await Promise.all([
        this.getUser(userId),
        provider.userManagement.listOrganizationMemberships({
          userId,
          limit: 100,
          statuses: ["active"],
        }),
      ]);
      if (!user) {
        return { data: [], metadata: {} };
      }
      return {
        data: memberships.data.map((membership) => ({
          id: membership.id,
          user,
          organization: {
            id: membership.organizationId,
            name: membership.organizationName,
          },
          role: membership.role.slug,
          createdAt: membership.createdAt,
          updatedAt: membership.updatedAt,
          status: membership.status,
        })),
        metadata: memberships.listMetadata ?? {},
      };
    } catch (error) {
      throw this.providerError(error);
    }
  }

  async getOrganizationMemberList(orgId: string): Promise<MembershipListResponse> {
    if (!orgId) {
      throw new Error("Organization id is required.");
    }
    try {
      const provider = await this.getProvider();
      const [organization, members, users] = await Promise.all([
        this.getOrg(orgId),
        provider.userManagement.listOrganizationMemberships({
          organizationId: orgId,
          limit: 100,
          statuses: ["active"],
        }),
        provider.userManagement.listUsers({ organizationId: orgId, limit: 100 }),
      ]);
      const usersById = new Map(users.data.map((user) => [user.id, mapWorkOSUser(user)]));
      return {
        data: members.data.map((member) => {
          const user = usersById.get(member.userId);
          if (!user) {
            throw new Error(`User ${member.userId} not found`);
          }
          return {
            id: member.id,
            user,
            organization,
            role: member.role.slug,
            createdAt: member.createdAt,
            updatedAt: member.updatedAt,
            status: member.status,
          };
        }),
        metadata: members.listMetadata ?? {},
      };
    } catch (error) {
      throw this.providerError(error);
    }
  }

  async deactivateMembership(membershipId: string, orgId: string): Promise<void> {
    if (!membershipId || !orgId) {
      throw new Error("Membership id and organization id are required.");
    }
    try {
      await this.assertMembershipInOrg(membershipId, orgId);
      const provider = await this.getProvider();
      await provider.userManagement.deactivateOrganizationMembership(membershipId);
    } catch (error) {
      this.rethrowScopeError(error);
    }
  }

  private async assertMembershipInOrg(membershipId: string, orgId: string): Promise<void> {
    try {
      const provider = await this.getProvider();
      const membership = await provider.userManagement.getOrganizationMembership(membershipId);
      if (membership.organizationId !== orgId) {
        throw new OrganizationScopeError("membership", membershipId);
      }
    } catch (error) {
      if (error instanceof OrganizationScopeError) {
        throw error;
      }
      if (this.isNotFound(error)) {
        throw new OrganizationScopeError("membership", membershipId);
      }
      throw error;
    }
  }

  private isNotFound(error: unknown): boolean {
    return typeof error === "object" && error !== null && "status" in error && error.status === 404;
  }

  private rethrowScopeError(error: unknown): never {
    if (error instanceof OrganizationScopeError) {
      throw error;
    }
    throw this.providerError(error);
  }

  private transformOrganization(organization: ProviderOrganization): Organization {
    return {
      id: organization.id,
      name: organization.name,
      createdAt: organization.createdAt,
      updatedAt: organization.updatedAt,
    };
  }
}
