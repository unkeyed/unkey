import { describe, expect, it } from "vitest";
import {
  Audience,
  type Destination,
  type Member,
  type Rule,
  WORKSPACE_WIDE,
  emailRoutesForMember,
  resolveRecipients,
  ruleMatches,
  ruleReach,
  rulesCoveringResource,
  setEmailPreference,
  typeHasSendableRecipients,
  wasDeduplicated,
  withoutDestination,
} from "./notification-model";

const admin: Member = {
  userId: "user_admin",
  name: "Admin",
  email: "admin@example.com",
  role: "admin",
  billingContact: true,
};
const member: Member = {
  userId: "user_member",
  name: "Member",
  email: "member@example.com",
  role: "member",
  billingContact: false,
};
const discord: Destination = {
  id: "dst_discord",
  name: "#incidents",
  enabled: true,
  config: { channel: "discord", urlHint: "https://discord.com/api/webhooks/…/abcd" },
};

function rule(overrides: Partial<Rule>): Rule {
  return {
    id: "rule",
    name: "Rule",
    notificationType: "deployment.failed",
    scope: WORKSPACE_WIDE,
    enabled: true,
    targets: [],
    ...overrides,
  };
}

const failedInProd = {
  notificationType: "deployment.failed",
  resource: { projectId: "proj_a", appId: "app_a", environmentId: "env_prod" },
};

describe("ruleMatches", () => {
  it("matches workspace-wide rules for any resource", () => {
    expect(ruleMatches(rule({}), failedInProd)).toBe(true);
  });

  it("requires every set scope field to match", () => {
    const scoped = rule({ scope: { projectId: "proj_a", appId: "app_a", environmentId: null } });
    expect(ruleMatches(scoped, failedInProd)).toBe(true);
    expect(
      ruleMatches(scoped, { ...failedInProd, resource: { ...failedInProd.resource, appId: "b" } }),
    ).toBe(false);
  });

  it("skips disabled rules and other types", () => {
    expect(ruleMatches(rule({ enabled: false }), failedInProd)).toBe(false);
    expect(ruleMatches(rule({ notificationType: "billing.paused" }), failedInProd)).toBe(false);
  });
});

describe("resolveRecipients", () => {
  const members = [admin, member];

  it("deduplicates a member reached by an audience and by name across rules", () => {
    const resolved = resolveRecipients(
      {
        members,
        destinations: [discord],
        preferences: [],
        rules: [
          rule({
            id: "r1",
            name: "Admins",
            targets: [{ kind: "audience", audience: Audience.WORKSPACE_ADMINS }],
          }),
          rule({
            id: "r2",
            name: "Named",
            scope: { projectId: "proj_a", appId: null, environmentId: null },
            targets: [
              { kind: "member", userId: admin.userId },
              { kind: "destination", destinationId: discord.id },
            ],
          }),
          rule({
            id: "r3",
            name: "Also discord",
            targets: [{ kind: "destination", destinationId: discord.id }],
          }),
        ],
      },
      failedInProd,
    );

    expect(resolved.email).toHaveLength(1);
    expect(resolved.email[0].member.userId).toBe(admin.userId);
    expect(resolved.email[0].reasons).toEqual([
      { ruleId: "r1", ruleName: "Admins", via: ["Workspace admins"] },
      { ruleId: "r2", ruleName: "Named", via: ["Added by name"] },
    ]);
    expect(resolved.destinations).toHaveLength(1);
    expect(resolved.destinations[0].reasons.map((reason) => reason.ruleId)).toEqual(["r2", "r3"]);
  });

  it("merges targets of one rule that reach the same member", () => {
    const resolved = resolveRecipients(
      {
        members,
        destinations: [],
        preferences: [],
        rules: [
          rule({
            id: "r1",
            targets: [
              { kind: "audience", audience: Audience.WORKSPACE_ADMINS },
              { kind: "member", userId: admin.userId },
            ],
          }),
        ],
      },
      failedInProd,
    );
    expect(resolved.email[0].reasons).toEqual([
      { ruleId: "r1", ruleName: "Rule", via: ["Workspace admins", "Added by name"] },
    ]);
    expect(wasDeduplicated(resolved.email[0].reasons)).toBe(true);
  });

  it("marks opted-out members and paused destinations without dropping them", () => {
    const resolved = resolveRecipients(
      {
        members,
        destinations: [{ ...discord, enabled: false }],
        preferences: [
          { userId: member.userId, notificationType: "deployment.failed", enabled: false },
        ],
        rules: [
          rule({
            targets: [
              { kind: "member", userId: member.userId },
              { kind: "destination", destinationId: discord.id },
            ],
          }),
        ],
      },
      failedInProd,
    );

    expect(resolved.email[0].status).toBe("opted_out");
    expect(resolved.destinations[0].status).toBe("paused");
  });

  it("resolves billing contacts from members", () => {
    const resolved = resolveRecipients(
      {
        members,
        destinations: [],
        preferences: [],
        rules: [
          rule({
            notificationType: "billing.paused",
            targets: [{ kind: "audience", audience: Audience.BILLING_CONTACTS }],
          }),
        ],
      },
      { notificationType: "billing.paused", resource: {} },
    );
    expect(resolved.email.map((recipient) => recipient.member.userId)).toEqual([admin.userId]);
  });
});

describe("ruleReach", () => {
  it("counts recipients of a disabled scoped rule", () => {
    const reach = ruleReach(
      rule({
        enabled: false,
        scope: { projectId: "proj_a", appId: "app_a", environmentId: "env_prod" },
        targets: [{ kind: "audience", audience: Audience.WORKSPACE_ADMINS }],
      }),
      { members: [admin, member], destinations: [], preferences: [] },
    );
    expect(reach.email).toHaveLength(1);
  });
});

describe("typeHasSendableRecipients", () => {
  const members = [admin, member];
  const paused = { ...discord, enabled: false };

  it("is false when the only rule targets a paused destination", () => {
    expect(
      typeHasSendableRecipients(
        {
          members,
          destinations: [paused],
          preferences: [],
          rules: [rule({ targets: [{ kind: "destination", destinationId: paused.id }] })],
        },
        "deployment.failed",
      ),
    ).toBe(false);
  });

  it("is false when every email recipient opted out", () => {
    expect(
      typeHasSendableRecipients(
        {
          members,
          destinations: [],
          preferences: [
            { userId: admin.userId, notificationType: "deployment.failed", enabled: false },
          ],
          rules: [rule({ targets: [{ kind: "audience", audience: Audience.WORKSPACE_ADMINS }] })],
        },
        "deployment.failed",
      ),
    ).toBe(false);
  });

  it("ignores disabled rules and counts scoped rules in their own scope", () => {
    const config = {
      members,
      destinations: [discord],
      preferences: [],
      rules: [
        rule({ enabled: false, targets: [{ kind: "member" as const, userId: member.userId }] }),
        rule({
          scope: { projectId: "proj_a", appId: null, environmentId: null },
          targets: [{ kind: "destination" as const, destinationId: discord.id }],
        }),
      ],
    };
    expect(typeHasSendableRecipients(config, "deployment.failed")).toBe(true);
    expect(typeHasSendableRecipients(config, "billing.paused")).toBe(false);
  });
});

describe("emailRoutesForMember", () => {
  it("lists each enabled rule once with the first matching target", () => {
    const routes = emailRoutesForMember(
      {
        members: [admin, member],
        destinations: [],
        preferences: [],
        rules: [
          rule({
            id: "r1",
            targets: [
              { kind: "audience", audience: Audience.WORKSPACE_ADMINS },
              { kind: "member", userId: admin.userId },
            ],
          }),
          rule({ id: "r2", enabled: false, targets: [{ kind: "member", userId: admin.userId }] }),
        ],
      },
      admin.userId,
      "deployment.failed",
    );
    expect(routes.map((route) => [route.rule.id, route.via])).toEqual([["r1", "Workspace admins"]]);
  });
});

describe("rulesCoveringResource", () => {
  const rules = [
    rule({ id: "workspace" }),
    rule({ id: "project", scope: { projectId: "proj_a", appId: null, environmentId: null } }),
    rule({ id: "app", scope: { projectId: "proj_a", appId: "app_a", environmentId: null } }),
    rule({ id: "other-app", scope: { projectId: "proj_a", appId: "app_b", environmentId: null } }),
    rule({ id: "other", scope: { projectId: "proj_b", appId: null, environmentId: null } }),
  ];

  it("splits project rules from workspace rules on a project", () => {
    const { scoped, inherited } = rulesCoveringResource(rules, { projectId: "proj_a" });
    expect(scoped.map((r) => r.id)).toEqual(["project", "app", "other-app"]);
    expect(inherited.map((r) => r.id)).toEqual(["workspace"]);
  });

  it("splits app rules from project and workspace rules on an app", () => {
    const { scoped, inherited } = rulesCoveringResource(rules, {
      projectId: "proj_a",
      appId: "app_a",
    });
    expect(scoped.map((r) => r.id)).toEqual(["app"]);
    expect(inherited.map((r) => r.id)).toEqual(["workspace", "project"]);
  });
});

describe("rule and preference updates", () => {
  it("removes a deleted destination from every rule", () => {
    const [updated] = withoutDestination(
      [
        rule({
          targets: [
            { kind: "destination", destinationId: discord.id },
            { kind: "member", userId: admin.userId },
          ],
        }),
      ],
      discord.id,
    );
    expect(updated.targets).toEqual([{ kind: "member", userId: admin.userId }]);
  });

  it("replaces an existing preference row", () => {
    const first = setEmailPreference([], {
      userId: admin.userId,
      notificationType: "billing.paused",
      enabled: false,
    });
    const second = setEmailPreference(first, {
      userId: admin.userId,
      notificationType: "billing.paused",
      enabled: true,
    });
    expect(second).toEqual([
      { userId: admin.userId, notificationType: "billing.paused", enabled: true },
    ]);
  });
});
