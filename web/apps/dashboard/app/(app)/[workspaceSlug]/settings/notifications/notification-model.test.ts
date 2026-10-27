import { describe, expect, it } from "vitest";
import {
  type Rule,
  WORKSPACE_WIDE,
  ruleFields,
  rulesCoveringResource,
  rulesUsingDestination,
  toRuleModel,
} from "./notification-model";
import { supportedNotificationTypes } from "./notification-types";

function rule(overrides: Partial<Rule>): Rule {
  return {
    id: crypto.randomUUID(),
    name: "Rule",
    notificationType: "deployment.failed",
    scope: WORKSPACE_WIDE,
    enabled: true,
    targets: [],
    ...overrides,
  };
}

describe("rule replacement", () => {
  it("preserves all fields when toggling a scoped rule", () => {
    const input = {
      id: crypto.randomUUID(),
      name: "Failures",
      notificationType: "deployment.failed",
      projectId: crypto.randomUUID(),
      appId: crypto.randomUUID(),
      environmentId: crypto.randomUUID(),
      enabled: true,
      targets: [{ type: "audience" as const, audience: "workspace_admins" as const }],
    };
    const { id, ...fields } = input;
    expect(ruleFields({ ...toRuleModel(input), enabled: false })).toEqual({
      ...fields,
      enabled: false,
    });
  });

  it("omits old scope fields when a full replacement broadens to the workspace", () => {
    const scoped = rule({
      scope: {
        projectId: crypto.randomUUID(),
        appId: crypto.randomUUID(),
        environmentId: crypto.randomUUID(),
      },
    });
    const replacement = ruleFields({ ...scoped, scope: WORKSPACE_WIDE });
    expect(JSON.parse(JSON.stringify(replacement))).toEqual({
      name: scoped.name,
      notificationType: scoped.notificationType,
      enabled: true,
      targets: [],
    });
    expect(toRuleModel({ id: scoped.id, ...replacement }).scope).toEqual(WORKSPACE_WIDE);
  });

  it("finds references without inventing recipient eligibility", () => {
    const destinationId = crypto.randomUUID();
    const targeted = rule({ enabled: false, targets: [{ type: "destination", destinationId }] });
    expect(rulesUsingDestination([targeted, rule({})], destinationId)).toEqual([targeted]);
  });
});

describe("rulesCoveringResource", () => {
  const projectId = crypto.randomUUID();
  const appId = crypto.randomUUID();
  const environmentId = crypto.randomUUID();
  const workspace = rule({});
  const project = rule({ scope: { projectId, appId: null, environmentId: null } });
  const app = rule({ scope: { projectId, appId, environmentId: null } });
  const siblingApp = rule({
    scope: { projectId, appId: crypto.randomUUID(), environmentId: null },
  });
  const otherProject = rule({
    scope: { projectId: crypto.randomUUID(), appId: null, environmentId: null },
  });
  const rules = [workspace, project, app, siblingApp, otherProject];

  it("splits project routes from broader workspace routes", () => {
    expect(rulesCoveringResource(rules, { projectId })).toEqual({
      scoped: [project, app, siblingApp],
      inherited: [workspace],
    });
  });

  it("splits app routes from broader routes and excludes sibling apps", () => {
    expect(rulesCoveringResource(rules, { projectId, appId })).toEqual({
      scoped: [app],
      inherited: [workspace, project],
    });
  });

  it("excludes sibling environments without excluding paused matching rules", () => {
    const production = rule({ enabled: false, scope: { projectId, appId, environmentId } });
    const preview = rule({ scope: { projectId, appId, environmentId: crypto.randomUUID() } });
    expect(
      rulesCoveringResource([...rules, production, preview], { projectId, appId, environmentId }),
    ).toEqual({ scoped: [production], inherited: [workspace, project, app] });
  });
});

it("only offers deployment types advertised by the server", () => {
  expect(
    supportedNotificationTypes(["deployment.failed", "billing.paused", "unknown.event"]).map(
      (type) => type.type,
    ),
  ).toEqual(["deployment.failed"]);
  expect(supportedNotificationTypes([])).toEqual([]);
});
