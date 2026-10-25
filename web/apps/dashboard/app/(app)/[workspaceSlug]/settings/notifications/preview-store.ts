import { z } from "zod";
import {
  Audience,
  type Destination,
  type EmailPreference,
  type Member,
  type Rule,
  WORKSPACE_WIDE,
  setEmailPreference,
  withoutDestination,
} from "./notification-model";

const STORAGE_PREFIX = "unkey.notifications-preview.v1";

const scopeSchema = z.object({
  projectId: z.string().nullable(),
  appId: z.string().nullable(),
  environmentId: z.string().nullable(),
});

const audienceSchema = z.union([
  z.literal(Audience.WORKSPACE_ADMINS),
  z.literal(Audience.BILLING_CONTACTS),
]);

const previewStateSchema = z.object({
  destinations: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      enabled: z.boolean(),
      config: z.discriminatedUnion("channel", [
        z.object({ channel: z.literal("discord"), urlHint: z.string() }),
        z.object({ channel: z.literal("slack"), urlHint: z.string() }),
        z.object({
          channel: z.literal("webhook"),
          urlHint: z.string(),
          headerNames: z.array(z.string()),
          signingSecretHint: z.string(),
        }),
      ]),
    }),
  ),
  rules: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      notificationType: z.string(),
      scope: scopeSchema,
      enabled: z.boolean(),
      targets: z.array(
        z.discriminatedUnion("kind", [
          z.object({ kind: z.literal("audience"), audience: audienceSchema }),
          z.object({ kind: z.literal("member"), userId: z.string() }),
          z.object({ kind: z.literal("destination"), destinationId: z.string() }),
        ]),
      ),
    }),
  ),
  preferences: z.array(
    z.object({ userId: z.string(), notificationType: z.string(), enabled: z.boolean() }),
  ),
});

export type PreviewState = {
  destinations: Destination[];
  rules: Rule[];
  preferences: EmailPreference[];
};

export const SAMPLE_MEMBERS: ReadonlyArray<Member> = [
  {
    userId: "user_sample_priya",
    name: "Priya Raman",
    email: "priya@acme.dev",
    role: "admin",
    billingContact: false,
  },
  {
    userId: "user_sample_marco",
    name: "Marco Silva",
    email: "marco@acme.dev",
    role: "member",
    billingContact: true,
  },
  {
    userId: "user_sample_ada",
    name: "Ada Okafor",
    email: "ada@acme.dev",
    role: "member",
    billingContact: false,
  },
];

const SEED: PreviewState = {
  destinations: [
    {
      id: "dst_preview_incidents",
      name: "#incidents",
      enabled: true,
      config: { channel: "discord", urlHint: "discord.com/api/webhooks/…Qx7s" },
    },
    {
      id: "dst_preview_billing",
      name: "#billing-alerts",
      enabled: true,
      config: { channel: "slack", urlHint: "hooks.slack.com/services/…9kLm" },
    },
    {
      id: "dst_preview_oncall",
      name: "On-call bridge",
      enabled: false,
      config: {
        channel: "webhook",
        urlHint: "oncall.acme.dev/hooks/…/unkey",
        headerNames: ["X-Routing-Key"],
        signingSecretHint: "whsec_…4f2a",
      },
    },
  ],
  rules: [
    {
      id: "rule_preview_deploy_failed",
      name: "Deploy failures to on-call",
      notificationType: "deployment.failed",
      scope: WORKSPACE_WIDE,
      enabled: true,
      targets: [
        { kind: "audience", audience: Audience.WORKSPACE_ADMINS },
        { kind: "destination", destinationId: "dst_preview_incidents" },
        { kind: "destination", destinationId: "dst_preview_oncall" },
      ],
    },
    {
      id: "rule_preview_rollbacks",
      name: "Rollbacks in #incidents",
      notificationType: "deployment.rolled_back",
      scope: WORKSPACE_WIDE,
      enabled: true,
      targets: [{ kind: "destination", destinationId: "dst_preview_incidents" }],
    },
    {
      id: "rule_preview_payment_failed",
      name: "Payment problems",
      notificationType: "billing.payment_failed",
      scope: WORKSPACE_WIDE,
      enabled: true,
      targets: [
        { kind: "audience", audience: Audience.BILLING_CONTACTS },
        { kind: "audience", audience: Audience.WORKSPACE_ADMINS },
        { kind: "destination", destinationId: "dst_preview_billing" },
      ],
    },
    {
      id: "rule_preview_paused",
      name: "Workspace paused",
      notificationType: "billing.paused",
      scope: WORKSPACE_WIDE,
      enabled: true,
      targets: [
        { kind: "audience", audience: Audience.WORKSPACE_ADMINS },
        { kind: "member", userId: "user_sample_marco" },
        { kind: "destination", destinationId: "dst_preview_billing" },
      ],
    },
  ],
  preferences: [
    { userId: "user_sample_priya", notificationType: "billing.payment_failed", enabled: false },
  ],
};

export type PreviewStorage = Pick<Storage, "getItem" | "setItem">;

export type PreviewActions = {
  saveRule: (rule: Rule) => void;
  setRuleEnabled: (id: string, enabled: boolean) => void;
  deleteRule: (id: string) => void;
  saveDestination: (destination: Destination) => void;
  setDestinationEnabled: (id: string, enabled: boolean) => void;
  deleteDestination: (id: string) => void;
  setEmailPreference: (preference: EmailPreference) => void;
  reset: () => void;
};

export type PreviewStore = {
  workspaceId: string;
  subscribe: (listener: () => void) => () => void;
  getSnapshot: () => PreviewState;
  getServerSnapshot: () => PreviewState;
  actions: PreviewActions;
};

export function previewStorageKey(workspaceId: string): string {
  return `${STORAGE_PREFIX}.${workspaceId}`;
}

export function createPreviewStore(
  workspaceId: string,
  getStorage: () => PreviewStorage | null,
): PreviewStore {
  const key = previewStorageKey(workspaceId);
  const listeners = new Set<() => void>();
  let current: PreviewState | null = null;

  function readStored(): unknown {
    try {
      const raw = getStorage()?.getItem(key);
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  }

  function load(): PreviewState {
    const parsed = previewStateSchema.safeParse(readStored());
    return parsed.success ? parsed.data : SEED;
  }

  function persist(state: PreviewState): boolean {
    try {
      getStorage()?.setItem(key, JSON.stringify(state));
      return true;
    } catch {
      return false;
    }
  }

  function getSnapshot(): PreviewState {
    current ??= load();
    return current;
  }

  function update(next: (state: PreviewState) => PreviewState) {
    current = next(getSnapshot());
    persist(current);
    for (const listener of listeners) {
      listener();
    }
  }

  return {
    workspaceId,
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    getSnapshot,
    getServerSnapshot: () => SEED,
    actions: {
      saveRule(rule) {
        update((state) => ({
          ...state,
          rules: state.rules.some((existing) => existing.id === rule.id)
            ? state.rules.map((existing) => (existing.id === rule.id ? rule : existing))
            : [...state.rules, rule],
        }));
      },
      setRuleEnabled(id, enabled) {
        update((state) => ({
          ...state,
          rules: state.rules.map((rule) => (rule.id === id ? { ...rule, enabled } : rule)),
        }));
      },
      deleteRule(id) {
        update((state) => ({ ...state, rules: state.rules.filter((rule) => rule.id !== id) }));
      },
      saveDestination(destination) {
        update((state) => ({
          ...state,
          destinations: state.destinations.some((existing) => existing.id === destination.id)
            ? state.destinations.map((existing) =>
                existing.id === destination.id ? destination : existing,
              )
            : [...state.destinations, destination],
        }));
      },
      setDestinationEnabled(id, enabled) {
        update((state) => ({
          ...state,
          destinations: state.destinations.map((destination) =>
            destination.id === id ? { ...destination, enabled } : destination,
          ),
        }));
      },
      deleteDestination(id) {
        update((state) => ({
          ...state,
          destinations: state.destinations.filter((destination) => destination.id !== id),
          rules: withoutDestination(state.rules, id),
        }));
      },
      setEmailPreference(preference) {
        update((state) => ({
          ...state,
          preferences: setEmailPreference(state.preferences, preference),
        }));
      },
      reset() {
        update(() => SEED);
      },
    },
  };
}

const browserStores = new Map<string, PreviewStore>();

function browserSessionStorage(): PreviewStorage | null {
  return typeof window === "undefined" ? null : window.sessionStorage;
}

export function previewStoreFor(workspaceId: string): PreviewStore {
  let store = browserStores.get(workspaceId);
  if (!store) {
    store = createPreviewStore(workspaceId, browserSessionStorage);
    browserStores.set(workspaceId, store);
  }
  return store;
}

export function newPreviewId(prefix: "rule" | "dst"): string {
  return `${prefix}_preview_${crypto.randomUUID().slice(0, 8)}`;
}

export function isSampleMember(userId: string): boolean {
  return SAMPLE_MEMBERS.some((member) => member.userId === userId);
}
