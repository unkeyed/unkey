// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/db", () => ({
  primaryDb: {},
}));

import { LocalAuthProvider } from "@/lib/auth/local";
import { LOCAL_ORG_ID } from "@/lib/auth/types";
import { createMembershipOrDeleteOrg } from "@/lib/auth/workos";
import {
  type WorkspaceCreateDeps,
  WorkspaceCreateError,
  createFreeWorkspace,
} from "./create-workspace";

const input = {
  name: "Acme",
  slug: "acme",
  userId: "user_abc",
  audit: { location: "127.0.0.1" },
  localOrgId: null,
};

function deps(overrides: Partial<WorkspaceCreateDeps> = {}): WorkspaceCreateDeps {
  return {
    findWorkspacesByOrgId: vi.fn(async () => []),
    findWorkspaceBySlug: vi.fn(async () => undefined),
    persist: vi.fn(async (created) => ({
      orgId: created.orgId,
      workspaceId: "ws_123",
      slug: created.slug,
    })),
    ...overrides,
  };
}

describe("createFreeWorkspace", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("creates the organization before the database write and keeps it on success", async () => {
    const createTenant = vi.fn(async () => "org_123");
    const deleteTenant = vi.fn(async () => undefined);
    const store = deps();
    await expect(
      createFreeWorkspace(
        {
          ...input,
          metadata: { agent_registration_id: "agent_reg_abc" },
          createTenant,
          deleteTenant,
        },
        store,
      ),
    ).resolves.toEqual({ orgId: "org_123", workspaceId: "ws_123", slug: "acme" });
    expect(createTenant).toHaveBeenCalledWith({
      name: "Acme",
      userId: "user_abc",
      metadata: { agent_registration_id: "agent_reg_abc" },
    });
    expect(store.persist).toHaveBeenCalledWith(expect.objectContaining({ orgId: "org_123" }));
    expect(deleteTenant).not.toHaveBeenCalled();
  });

  it("does not create an organization when the slug is already taken", async () => {
    const createTenant = vi.fn(async () => "org_123");
    const deleteTenant = vi.fn(async () => undefined);
    await expect(
      createFreeWorkspace(
        { ...input, createTenant, deleteTenant },
        deps({ findWorkspaceBySlug: vi.fn(async () => ({ id: "ws_existing" })) }),
      ),
    ).rejects.toMatchObject({ code: "CONFLICT" });
    expect(createTenant).not.toHaveBeenCalled();
    expect(deleteTenant).not.toHaveBeenCalled();
  });

  it("deletes the organization when the database write fails", async () => {
    const createTenant = vi.fn(async () => "org_orphan");
    const deleteTenant = vi.fn(async () => undefined);
    const failure = new WorkspaceCreateError(
      "CONFLICT",
      "A workspace with this slug already exists.",
    );
    await expect(
      createFreeWorkspace(
        { ...input, createTenant, deleteTenant },
        deps({
          persist: vi.fn(async () => {
            throw failure;
          }),
        }),
      ),
    ).rejects.toBe(failure);
    expect(deleteTenant).toHaveBeenCalledWith("org_orphan");
  });

  it("still throws the original error when organization cleanup fails", async () => {
    const createTenant = vi.fn(async () => "org_orphan");
    const deleteTenant = vi.fn(async () => {
      throw new Error("workos down");
    });
    const failure = new Error("insert failed");
    await expect(
      createFreeWorkspace(
        { ...input, createTenant, deleteTenant },
        deps({
          persist: vi.fn(async () => {
            throw failure;
          }),
        }),
      ),
    ).rejects.toBe(failure);
    expect(deleteTenant).toHaveBeenCalledWith("org_orphan");
  });

  it("does not delete an organization when creation itself fails", async () => {
    const deleteTenant = vi.fn(async () => undefined);
    await expect(
      createFreeWorkspace(
        {
          ...input,
          createTenant: vi.fn(async () => {
            throw new Error("workos rejected");
          }),
          deleteTenant,
        },
        deps(),
      ),
    ).rejects.toThrow("workos rejected");
    expect(deleteTenant).not.toHaveBeenCalled();
  });
});

describe("organization cleanup", () => {
  it("deletes the organization when membership creation fails", async () => {
    const deleteOrganization = vi.fn(async () => undefined);
    const failure = new Error("membership failed");
    await expect(
      createMembershipOrDeleteOrg({
        createMembership: async () => {
          throw failure;
        },
        deleteOrganization,
      }),
    ).rejects.toBe(failure);
    expect(deleteOrganization).toHaveBeenCalledOnce();
  });

  it("leaves the local organization in place", async () => {
    const provider = new LocalAuthProvider();
    await expect(provider.deleteTenant(LOCAL_ORG_ID)).resolves.toBeUndefined();
    await expect(provider.createTenant({ name: "Acme", userId: "user_local_admin" })).resolves.toBe(
      LOCAL_ORG_ID,
    );
  });
});
