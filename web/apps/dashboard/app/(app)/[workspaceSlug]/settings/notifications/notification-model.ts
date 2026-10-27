import type {
  NotificationAudienceTarget,
  NotificationChannel,
  NotificationDestination,
  NotificationMember,
  NotificationRule,
  NotificationTarget,
  V2NotificationsUpdateRuleRequestBody,
} from "@unkey/api/models/components";

export type NamedAudience = NotificationAudienceTarget["audience"];
export type DestinationChannel = NotificationChannel;
export type Destination = NotificationDestination;
export type Member = NotificationMember;
export type RecipientTarget = NotificationTarget;

export const NAMED_AUDIENCES: ReadonlyArray<{
  audience: NamedAudience;
  label: string;
  description: string;
}> = [
  {
    audience: "workspace_admins",
    label: "Workspace admins",
    description: "Every member with the admin role.",
  },
];

export function audienceLabel(audience: NamedAudience): string {
  return NAMED_AUDIENCES.find((option) => option.audience === audience)?.label ?? audience;
}

export type ResourceScope = {
  projectId: string | null;
  appId: string | null;
  environmentId: string | null;
};

export const WORKSPACE_WIDE: ResourceScope = { projectId: null, appId: null, environmentId: null };

export type Rule = Omit<NotificationRule, "projectId" | "appId" | "environmentId"> & {
  scope: ResourceScope;
};

export function toRuleModel(rule: NotificationRule): Rule {
  const { projectId, appId, environmentId, ...fields } = rule;
  return {
    ...fields,
    scope: {
      projectId: projectId ?? null,
      appId: appId ?? null,
      environmentId: environmentId ?? null,
    },
  };
}

export function ruleFields(
  rule: Omit<Rule, "id">,
): Omit<V2NotificationsUpdateRuleRequestBody, "ruleId"> {
  return {
    name: rule.name,
    notificationType: rule.notificationType,
    enabled: rule.enabled,
    targets: rule.targets,
    projectId: rule.scope.projectId ?? undefined,
    appId: rule.scope.appId ?? undefined,
    environmentId: rule.scope.environmentId ?? undefined,
  };
}

export function rulesUsingDestination(rules: ReadonlyArray<Rule>, destinationId: string): Rule[] {
  return rules.filter((rule) =>
    rule.targets.some(
      (target) => target.type === "destination" && target.destinationId === destinationId,
    ),
  );
}

export function rulesCoveringResource(
  rules: ReadonlyArray<Rule>,
  resource: { projectId: string; appId?: string; environmentId?: string },
): { scoped: Rule[]; inherited: Rule[] } {
  const scoped: Rule[] = [];
  const inherited: Rule[] = [];
  for (const rule of rules) {
    const { projectId, appId, environmentId } = rule.scope;
    if (projectId !== null && projectId !== resource.projectId) {
      continue;
    }
    if (resource.appId !== undefined && appId !== null && appId !== resource.appId) {
      continue;
    }
    if (
      resource.environmentId !== undefined &&
      environmentId !== null &&
      environmentId !== resource.environmentId
    ) {
      continue;
    }
    const isTargeted =
      resource.environmentId !== undefined
        ? environmentId !== null
        : resource.appId !== undefined
          ? appId !== null
          : projectId !== null;
    (isTargeted ? scoped : inherited).push(rule);
  }
  return { scoped, inherited };
}
