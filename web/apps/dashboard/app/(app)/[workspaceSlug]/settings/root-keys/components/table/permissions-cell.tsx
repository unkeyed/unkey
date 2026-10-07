import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { match } from "@unkey/match";
import { type SummaryTemplateId, permissionSummary } from "./permission-summary";

const TEMPLATE_LABELS: Record<SummaryTemplateId, string> = {
  read: "All read",
  write: "All write",
  verify: "Verify keys",
  ratelimit: "Ratelimiting",
};

type PermissionsCellProps = {
  permissions: { name: string }[];
};

export function PermissionsCell({ permissions }: PermissionsCellProps) {
  const workspace = useWorkspaceNavigation();
  const summary = permissionSummary(
    workspace.id,
    permissions.map((permission) => permission.name),
  );

  return match(summary)
    .with({ type: "none" }, () => <span className="text-sm leading-5 text-gray-8">—</span>)
    .with({ type: "template" }, ({ template }) => (
      <span className="truncate text-sm leading-5 text-gray-11">{TEMPLATE_LABELS[template]}</span>
    ))
    .with({ type: "restricted" }, () => (
      <span className="truncate text-sm leading-5 text-gray-11">Restricted</span>
    ))
    .exhaustive();
}
