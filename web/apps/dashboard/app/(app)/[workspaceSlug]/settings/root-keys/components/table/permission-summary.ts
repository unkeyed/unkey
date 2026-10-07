import { TEMPLATES, type TemplateId } from "../builder/lib/templates";
import { buildUrns } from "../builder/lib/urn";

export type SummaryTemplateId = Exclude<TemplateId, "custom">;

export type PermissionSummary =
  | { type: "none" }
  | { type: "template"; template: SummaryTemplateId }
  | { type: "restricted" };

function isSummaryTemplate(id: TemplateId): id is SummaryTemplateId {
  return id !== "custom";
}

function sameGrants(left: ReadonlySet<string>, right: readonly string[]): boolean {
  return left.size === right.length && right.every((grant) => left.has(grant));
}

export function permissionSummary(
  workspaceId: string,
  grants: readonly string[],
): PermissionSummary {
  if (grants.length === 0) {
    return { type: "none" };
  }
  const held = new Set(grants);
  for (const template of TEMPLATES) {
    if (
      isSummaryTemplate(template.id) &&
      sameGrants(held, buildUrns(workspaceId, template.materialise()))
    ) {
      return { type: "template", template: template.id };
    }
  }
  return { type: "restricted" };
}
