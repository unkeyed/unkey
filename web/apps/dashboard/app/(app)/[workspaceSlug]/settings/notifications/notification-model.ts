import { Audience } from "@unkey/db/src/gen/notification/v1/config_pb";

export { Audience };

export type NamedAudience = Audience.WORKSPACE_ADMINS | Audience.BILLING_CONTACTS;

export const NAMED_AUDIENCES: ReadonlyArray<{
  audience: NamedAudience;
  label: string;
  description: string;
}> = [
  {
    audience: Audience.WORKSPACE_ADMINS,
    label: "Workspace admins",
    description: "Every member with the admin role.",
  },
  {
    audience: Audience.BILLING_CONTACTS,
    label: "Billing contacts",
    description: "Members who receive invoices and payment notices.",
  },
];

export function audienceLabel(audience: NamedAudience): string {
  return audience === Audience.WORKSPACE_ADMINS ? "Workspace admins" : "Billing contacts";
}

export type DestinationChannel = "discord" | "slack" | "webhook";

export type DestinationConfigSummary =
  | { channel: "discord"; urlHint: string }
  | { channel: "slack"; urlHint: string }
  | {
      channel: "webhook";
      urlHint: string;
      headerNames: string[];
      signingSecretHint: string;
    };

export type Destination = {
  id: string;
  name: string;
  enabled: boolean;
  config: DestinationConfigSummary;
};

export type RecipientTarget =
  | { kind: "audience"; audience: NamedAudience }
  | { kind: "member"; userId: string }
  | { kind: "destination"; destinationId: string };

export type ResourceScope = {
  projectId: string | null;
  appId: string | null;
  environmentId: string | null;
};

export const WORKSPACE_WIDE: ResourceScope = { projectId: null, appId: null, environmentId: null };

export type Rule = {
  id: string;
  name: string;
  notificationType: string;
  scope: ResourceScope;
  enabled: boolean;
  targets: RecipientTarget[];
};

export type EmailPreference = {
  userId: string;
  notificationType: string;
  enabled: boolean;
};

export type Member = {
  userId: string;
  name: string;
  email: string;
  role: "admin" | "member";
  billingContact: boolean;
};

export type RoutingConfig = {
  rules: ReadonlyArray<Rule>;
  destinations: ReadonlyArray<Destination>;
  preferences: ReadonlyArray<EmailPreference>;
  members: ReadonlyArray<Member>;
};

export type NotificationEvent = {
  notificationType: string;
  resource: { projectId?: string; appId?: string; environmentId?: string };
};

export function ruleMatches(rule: Rule, event: NotificationEvent): boolean {
  if (!rule.enabled || rule.notificationType !== event.notificationType) {
    return false;
  }
  const { projectId, appId, environmentId } = rule.scope;
  return (
    (projectId === null || projectId === event.resource.projectId) &&
    (appId === null || appId === event.resource.appId) &&
    (environmentId === null || environmentId === event.resource.environmentId)
  );
}

export function audienceMembers(members: ReadonlyArray<Member>, audience: NamedAudience): Member[] {
  return members.filter((member) =>
    audience === Audience.WORKSPACE_ADMINS ? member.role === "admin" : member.billingContact,
  );
}

export function isEmailEnabled(
  preferences: ReadonlyArray<EmailPreference>,
  userId: string,
  notificationType: string,
): boolean {
  const preference = preferences.find(
    (candidate) => candidate.userId === userId && candidate.notificationType === notificationType,
  );
  return preference?.enabled ?? true;
}

export type RouteReason = { ruleId: string; ruleName: string; via: string[] };

function addReason(reasons: RouteReason[], rule: Rule, via: string) {
  const existing = reasons.find((reason) => reason.ruleId === rule.id);
  if (existing) {
    if (!existing.via.includes(via)) {
      existing.via.push(via);
    }
    return;
  }
  reasons.push({ ruleId: rule.id, ruleName: rule.name, via: [via] });
}

export function wasDeduplicated(reasons: ReadonlyArray<RouteReason>): boolean {
  return reasons.reduce((count, reason) => count + reason.via.length, 0) > 1;
}

export type EmailRecipient = {
  member: Member;
  reasons: RouteReason[];
  status: "send" | "opted_out";
};

export type DestinationRecipient = {
  destination: Destination;
  reasons: RouteReason[];
  status: "send" | "paused";
};

export type ResolvedRecipients = {
  email: EmailRecipient[];
  destinations: DestinationRecipient[];
};

function emailTargetsOf(
  target: RecipientTarget,
  members: ReadonlyArray<Member>,
): Array<{ member: Member; via: string }> {
  if (target.kind === "audience") {
    const via = audienceLabel(target.audience);
    return audienceMembers(members, target.audience).map((member) => ({ member, via }));
  }
  if (target.kind === "member") {
    const member = members.find((candidate) => candidate.userId === target.userId);
    return member ? [{ member, via: "Added by name" }] : [];
  }
  return [];
}

export function resolveRecipients(
  config: RoutingConfig,
  event: NotificationEvent,
): ResolvedRecipients {
  const email = new Map<string, EmailRecipient>();
  const destinations = new Map<string, DestinationRecipient>();

  for (const rule of config.rules) {
    if (!ruleMatches(rule, event)) {
      continue;
    }
    for (const target of rule.targets) {
      for (const { member, via } of emailTargetsOf(target, config.members)) {
        const existing = email.get(member.userId);
        if (existing) {
          addReason(existing.reasons, rule, via);
          continue;
        }
        email.set(member.userId, {
          member,
          reasons: [{ ruleId: rule.id, ruleName: rule.name, via: [via] }],
          status: isEmailEnabled(config.preferences, member.userId, event.notificationType)
            ? "send"
            : "opted_out",
        });
      }

      if (target.kind !== "destination") {
        continue;
      }
      const destination = config.destinations.find(
        (candidate) => candidate.id === target.destinationId,
      );
      if (!destination) {
        continue;
      }
      const existing = destinations.get(destination.id);
      if (existing) {
        addReason(existing.reasons, rule, destination.name);
        continue;
      }
      destinations.set(destination.id, {
        destination,
        reasons: [{ ruleId: rule.id, ruleName: rule.name, via: [destination.name] }],
        status: destination.enabled ? "send" : "paused",
      });
    }
  }

  return { email: [...email.values()], destinations: [...destinations.values()] };
}

export function ruleReach(rule: Rule, config: Omit<RoutingConfig, "rules">): ResolvedRecipients {
  return resolveRecipients(
    { ...config, rules: [{ ...rule, enabled: true }] },
    {
      notificationType: rule.notificationType,
      resource: {
        projectId: rule.scope.projectId ?? undefined,
        appId: rule.scope.appId ?? undefined,
        environmentId: rule.scope.environmentId ?? undefined,
      },
    },
  );
}

export function typeHasSendableRecipients(
  config: RoutingConfig,
  notificationType: string,
): boolean {
  return config.rules.some((rule) => {
    if (!rule.enabled || rule.notificationType !== notificationType) {
      return false;
    }
    const reach = ruleReach(rule, config);
    return (
      reach.email.some((recipient) => recipient.status === "send") ||
      reach.destinations.some((recipient) => recipient.status === "send")
    );
  });
}

export function emailRoutesForMember(
  config: RoutingConfig,
  userId: string,
  notificationType: string,
): Array<{ rule: Rule; via: string }> {
  const routes: Array<{ rule: Rule; via: string }> = [];
  for (const rule of config.rules) {
    if (!rule.enabled || rule.notificationType !== notificationType) {
      continue;
    }
    for (const target of rule.targets) {
      const match = emailTargetsOf(target, config.members).find(
        ({ member }) => member.userId === userId,
      );
      if (match) {
        routes.push({ rule, via: match.via });
        break;
      }
    }
  }
  return routes;
}

export function rulesUsingDestination(rules: ReadonlyArray<Rule>, destinationId: string): Rule[] {
  return rules.filter((rule) =>
    rule.targets.some(
      (target) => target.kind === "destination" && target.destinationId === destinationId,
    ),
  );
}

export function withoutDestination(rules: ReadonlyArray<Rule>, destinationId: string): Rule[] {
  return rules.map((rule) => ({
    ...rule,
    targets: rule.targets.filter(
      (target) => !(target.kind === "destination" && target.destinationId === destinationId),
    ),
  }));
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

export function setEmailPreference(
  preferences: ReadonlyArray<EmailPreference>,
  next: EmailPreference,
): EmailPreference[] {
  return [
    ...preferences.filter(
      (preference) =>
        !(
          preference.userId === next.userId && preference.notificationType === next.notificationType
        ),
    ),
    next,
  ];
}
