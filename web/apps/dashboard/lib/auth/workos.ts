import { logOperation } from "@/lib/logging";
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

type ProviderPage<T> = {
  data: T[];
  listMetadata?: { after?: string | null };
};

const PAGE_SIZE = 100;

/**
 * Bounds every cursor loop. A provider cursor that never settles would
 * otherwise hang the request forever instead of failing.
 */
const MAX_PAGES = 50;

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

  private async collectPages<T>(
    fetchPage: (after?: string) => Promise<ProviderPage<T>>,
  ): Promise<T[]> {
    const items: T[] = [];
    let after: string | undefined;

    for (let page = 0; page < MAX_PAGES; page++) {
      const result = await fetchPage(after);
      items.push(...result.data);
      after = result.listMetadata?.after ?? undefined;
      if (!after) {
        return items;
      }
    }

    throw new Error(`WorkOS list exceeded ${MAX_PAGES} pages`);
  }

  async getUser(userId: string): Promise<User | null> {
    if (!userId) {
      throw new Error("User Id is required.");
    }
    const provider = await this.getProvider();
    try {
      return mapWorkOSUser(await provider.userManagement.getUser(userId));
    } catch (error) {
      // Only a genuine 404 means "no such user". Collapsing transient provider
      // failures into null made an outage look like a deleted account, which
      // signed callers out instead of surfacing the failure.
      if (this.isNotFound(error)) {
        return null;
      }
      throw this.providerError(error);
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

  async listMemberships(userId: string, organizationId?: string): Promise<MembershipListResponse> {
    if (!userId) {
      throw new Error("User Id is required.");
    }
    try {
      const provider = await this.getProvider();
      const [user, memberships] = await Promise.all([
        this.getUser(userId),
        this.collectPages((after) =>
          provider.userManagement.listOrganizationMemberships({
            userId,
            organizationId,
            limit: PAGE_SIZE,
            statuses: ["active"],
            after,
          }),
        ),
      ]);
      if (!user) {
        return { data: [], metadata: {} };
      }
      return {
        data: memberships.map((membership) => ({
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
        metadata: {},
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
        this.collectPages((after) =>
          provider.userManagement.listOrganizationMemberships({
            organizationId: orgId,
            limit: PAGE_SIZE,
            statuses: ["active"],
            after,
          }),
        ),
        this.collectPages((after) =>
          provider.userManagement.listUsers({ organizationId: orgId, limit: PAGE_SIZE, after }),
        ),
      ]);
      const usersById = new Map(users.map((user) => [user.id, mapWorkOSUser(user)]));

      const unmatched = members.filter((member) => !usersById.has(member.userId));
      if (unmatched.length > 0) {
        const resolved = await Promise.all(
          unmatched.map(async (member) => await this.getUser(member.userId)),
        );
        for (const user of resolved) {
          if (user) {
            usersById.set(user.id, user);
          }
        }
      }

      return {
        data: members.flatMap((member) => {
          const user = usersById.get(member.userId);
          if (!user) {
            logOperation("warn", "WorkOS membership has no matching user", {
              auth_event: "member_list",
              org_id: orgId,
              membership_id: member.id,
            });
            return [];
          }
          return [
            {
              id: member.id,
              user,
              organization,
              role: member.role.slug,
              createdAt: member.createdAt,
              updatedAt: member.updatedAt,
              status: member.status,
            },
          ];
        }),
        metadata: {},
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
