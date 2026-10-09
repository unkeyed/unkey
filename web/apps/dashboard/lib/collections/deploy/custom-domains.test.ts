import { createLiveQueryCollection, eq } from "@tanstack/react-db";
import { describe, expect, it, vi } from "vitest";

const saved: { id: string; domain: string }[] = [];
const listDomains = vi.fn(async () => ({
  data: saved.map(({ id, domain }) => ({
    id,
    domain,
    projectId: "proj",
    appId: "app",
    environmentId: "env",
    status: "pending" as const,
    dnsRecords: [],
    createdAt: 0,
  })),
}));
const createDomain = vi.fn(async ({ domain }: { domain: string }) => {
  saved.push({ id: `dom_${domain}`, domain });
  return {};
});

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ domains: { listDomains, createDomain } }),
  getErrorMessage: () => "",
  getErrorToast: () => ({ message: "" }),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return { queryClient: new QueryClient() };
});

const { customDomains } = await import("./custom-domains");

describe("customDomains", () => {
  it("reloads the list once after adding a domain", async () => {
    const query = createLiveQueryCollection((q) =>
      q.from({ d: customDomains }).where(({ d }) => eq(d.projectId, "proj")),
    );
    await query.preload();
    listDomains.mockClear();

    const tx = customDomains.insert({
      id: "optimistic",
      domain: "api.acme.dev",
      projectId: "proj",
      appId: "app",
      environmentId: "env",
      verificationStatus: "pending",
      dnsRecords: [],
      verificationError: null,
      domainConnectProvider: null,
      domainConnectUrl: null,
      createdAt: 0,
      updatedAt: null,
    });
    await tx.isPersisted.promise;

    expect(listDomains).toHaveBeenCalledTimes(1);
    expect(query.toArray.map((d) => d.id)).toEqual(["dom_api.acme.dev"]);
  });
});
