import { newId } from "@unkey/id";
import { afterAll, expect, it, vi } from "vitest";
import { t } from "../trpc";
import { createPlainIssue } from "./plain";

const plain = vi.hoisted(() => {
  vi.stubEnv("UNKEY_WORKSPACE_ID", crypto.randomUUID());
  vi.stubEnv("UNKEY_API_ID", crypto.randomUUID());
  vi.stubEnv("UNKEY_ROOT_KEY", undefined);
  vi.stubEnv("VAULT_URL", "http://localhost");
  vi.stubEnv("VAULT_TOKEN", "test");
  vi.stubEnv("AUTH_PROVIDER", "local");
  vi.stubEnv("PLAIN_API_KEY", "test");
  return { createThread: vi.fn().mockResolvedValue({ error: null }) };
});

vi.mock("@team-plain/typescript-sdk", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@team-plain/typescript-sdk")>()),
  PlainClient: class {
    upsertCustomer = async () => ({ error: null });
    createThread = plain.createThread;
  },
}));

afterAll(() => vi.unstubAllEnvs());

it("attaches the page and authenticated workspace to feedback without URL secrets", async () => {
  const workspaceId = newId("workspace");
  const tenantId = newId("test");
  const userId = newId("test");
  const caller = t.router({ createPlainIssue }).createCaller({
    req: new Request("http://localhost/api/trpc"),
    audit: { location: "unknown", userAgent: undefined },
    user: {
      id: userId,
      profile: {
        id: userId,
        email: "user@example.com",
        firstName: null,
        lastName: null,
        fullName: null,
        avatarUrl: null,
      },
    },
    tenant: { id: tenantId, role: "admin" },
    workspace: {
      pk: 1,
      id: workspaceId,
      orgId: tenantId,
      name: "Feedback test",
      slug: "feedback-test",
      k8sNamespace: "feedback-test",
      betaFeatures: {},
      subscriptions: null,
      enabled: true,
      deleteProtection: false,
      createdAtM: Date.now(),
      updatedAtM: null,
      deletedAtM: null,
      limits: {
        pk: 1,
        workspaceId,
        apiBillableOperationsCountMaxPerMonth: 1000,
        apiRequestsCountMaxPerMinute: null,
        logsRetentionDaysMax: 7,
        logsAuditRetentionDaysMax: 7,
        logdrainsMax: 0,
        teamEnabled: true,
        cpuCoresMax: 1,
        cpuCoresMaxPerInstance: 1,
        memoryMibMax: 256,
        memoryMibMaxPerInstance: 256,
        storageMibMax: 256,
        storageMibMaxPerInstance: 256,
        buildsConcurrentMax: 1,
        customDomainsMax: 1,
        autoscalingReplicasMax: 1,
      },
      billing: null,
      billingSubscriptions: [],
      tier: "Free",
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      stripeDeploySubscriptionId: null,
      deployPlan: null,
      deployPlanOverride: null,
      deploySpendBudgetCents: null,
      deploySpendBudgetStop: false,
      deploySpendSuspended: false,
    },
  });
  const input = {
    issueType: "feedback" as const,
    severity: "p2" as const,
    message: "The usage chart does not show recent requests.",
    pageUrl: "https://app.unkey.com/team/apis?token=private-query#private-fragment",
    workspaceId: newId("workspace"),
  };

  await caller.createPlainIssue(input);

  const payload = JSON.stringify(plain.createThread.mock.calls[0]?.[0]);
  expect(payload).toContain("https://app.unkey.com/team/apis");
  expect(payload).toContain(workspaceId);
  expect(payload).toContain(tenantId);
  expect(payload).not.toContain(input.workspaceId);
  expect(payload).not.toContain("private-query");
  expect(payload).not.toContain("private-fragment");

  await expect(caller.createPlainIssue({ ...input, pageUrl: "not a URL" })).rejects.toMatchObject({
    code: "BAD_REQUEST",
  });
  await expect(
    caller.createPlainIssue({ ...input, pageUrl: "javascript:alert(1)" }),
  ).rejects.toMatchObject({ code: "BAD_REQUEST" });
  await expect(caller.createPlainIssue({ ...input, pageUrl: undefined })).resolves.toBeUndefined();
});
