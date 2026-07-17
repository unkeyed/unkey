/**
 * Recipe definitions for the Role Recipes concept: curated permission presets
 * where holes are typed placeholders. A hole resolves to the base resource
 * path of one concrete resource (e.g. the keyspace hole becomes
 * "keyspaces/ks_payments_prod"), and the rest of the template stays literal,
 * so filling one Select yields a full set of valid grants.
 */

import { KEYSPACES, PROJECTS, RATELIMIT_NAMESPACES } from "../lib/mock-data";

export type HoleKind = "keyspace" | "project" | "ratelimit_namespace";

export type TemplatePart = { kind: "literal"; value: string } | { kind: "hole"; hole: HoleKind };

export interface PermissionTemplate {
  /** resource-path fragments; joined with "/" once every hole is filled */
  parts: TemplatePart[];
  action: string;
}

export type RecipeCategory = "Observability" | "Operations" | "Deploy" | "Admin";

export interface Recipe {
  id: string;
  name: string;
  tagline: string;
  category: RecipeCategory;
  templates: PermissionTemplate[];
  /** warning copy for dangerous recipes */
  caution?: string;
  /** exact phrase the user must type before this recipe can be applied */
  confirmationPhrase?: string;
}

const lit = (value: string): TemplatePart => ({ kind: "literal", value });
const hole = (kind: HoleKind): TemplatePart => ({ kind: "hole", hole: kind });
const tpl = (parts: TemplatePart[], action: string): PermissionTemplate => ({ parts, action });

export const RECIPES: Recipe[] = [
  {
    id: "read-only-observer",
    name: "Read-only observer",
    tagline: "See every keyspace, key, and identity in the workspace without touching anything.",
    category: "Observability",
    templates: [
      tpl([lit("keyspaces/*")], "read_keyspace"),
      tpl([lit("keyspaces/*/keys/*")], "read_key"),
      tpl([lit("identities/*")], "read_identity"),
    ],
  },
  {
    id: "key-minting-service",
    name: "Key minting service",
    tagline: "Create new API keys in one keyspace and keep their metadata up to date.",
    category: "Operations",
    templates: [
      tpl([hole("keyspace")], "create_key"),
      tpl([hole("keyspace"), lit("keys/*")], "read_key"),
      tpl([hole("keyspace"), lit("keys/*")], "update_key"),
    ],
  },
  {
    id: "support-agent",
    name: "Support agent",
    tagline: "Look up customers and lift rate limits in one namespace when they call in.",
    category: "Operations",
    templates: [
      tpl([lit("identities/*")], "read_identity"),
      tpl([hole("ratelimit_namespace")], "set_override"),
    ],
  },
  {
    id: "ci-deployer",
    name: "CI deployer",
    tagline: "Ship new deployments and clean up old ones across one project.",
    category: "Deploy",
    templates: [
      tpl([hole("project"), lit("**")], "read_deployment"),
      tpl([hole("project"), lit("apps/*/environments/*")], "create_deployment"),
      tpl([hole("project"), lit("**")], "delete_deployment"),
    ],
  },
  {
    id: "key-rotation-bot",
    name: "Key rotation bot",
    tagline: "Rotate credentials on a schedule: mint replacements, then retire the old keys.",
    category: "Operations",
    templates: [
      tpl([hole("keyspace"), lit("keys/*")], "read_key"),
      tpl([hole("keyspace"), lit("keys/*")], "update_key"),
      tpl([hole("keyspace"), lit("keys/*")], "delete_key"),
      tpl([hole("keyspace")], "create_key"),
    ],
  },
  {
    id: "break-glass-admin",
    name: "Break-glass admin",
    tagline: "Every action on every resource. The admin escape hatch, for incidents only.",
    category: "Admin",
    templates: [tpl([lit("**")], "*")],
    caution:
      "This single grant covers every resource and every action in the workspace, including resources created in the future. Prefer a scoped recipe; reach for this one only during an incident.",
    confirmationPhrase: "admin",
  },
];

export interface HoleOption {
  /** base resource path this option resolves to, e.g. "keyspaces/ks_payments_prod" */
  path: string;
  /** human name, e.g. "payments-prod" */
  label: string;
}

export interface HoleMeta {
  kind: HoleKind;
  /** inline pill text, e.g. "{keyspace}" */
  placeholder: string;
  /** select label, e.g. "Keyspace" */
  selectLabel: string;
  options: HoleOption[];
}

export const HOLE_META: Record<HoleKind, HoleMeta> = {
  keyspace: {
    kind: "keyspace",
    placeholder: "{keyspace}",
    selectLabel: "Keyspace",
    options: KEYSPACES.map((ks) => ({ path: `keyspaces/${ks.id}`, label: ks.name })),
  },
  project: {
    kind: "project",
    placeholder: "{project}",
    selectLabel: "Project",
    options: PROJECTS.map((p) => ({ path: `projects/${p.id}`, label: p.name })),
  },
  ratelimit_namespace: {
    kind: "ratelimit_namespace",
    placeholder: "{namespace}",
    selectLabel: "Ratelimit namespace",
    options: RATELIMIT_NAMESPACES.map((ns) => ({
      path: `ratelimits/namespaces/${ns.id}`,
      label: ns.name,
    })),
  },
};

/** hole kind -> chosen base resource path */
export type HoleSelections = Partial<Record<HoleKind, string>>;

/** Distinct hole kinds in template order, one Select each. */
export function distinctHoles(recipe: Recipe): HoleKind[] {
  const seen: HoleKind[] = [];
  for (const template of recipe.templates) {
    for (const part of template.parts) {
      if (part.kind === "hole" && !seen.includes(part.hole)) {
        seen.push(part.hole);
      }
    }
  }
  return seen;
}

/** Resolved resource path, or null while any hole in the template is unfilled. */
export function resolveTemplatePath(
  template: PermissionTemplate,
  selections: HoleSelections,
): string | null {
  const fragments: string[] = [];
  for (const part of template.parts) {
    if (part.kind === "literal") {
      fragments.push(part.value);
      continue;
    }
    const chosen = selections[part.hole];
    if (chosen === undefined) {
      return null;
    }
    fragments.push(chosen);
  }
  return fragments.join("/");
}
