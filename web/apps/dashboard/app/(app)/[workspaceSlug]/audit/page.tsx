import { Navigation } from "@/components/navigation/navigation";
import { getAuth } from "@/lib/auth";
import { routes } from "@/lib/navigation/routes";
import { InputSearch } from "@unkey/icons";
import { getWorkspace } from "./actions";
import { LogsClient } from "./components/logs-client";
export const dynamic = "force-dynamic";

export default async function AuditPage() {
  const { orgId } = await getAuth();
  const { workspace, members } = await getWorkspace(orgId);

  return (
    <div>
      <Navigation
        href={routes.audit.list({ workspaceSlug: workspace.slug })}
        name="Audit"
        icon={<InputSearch />}
      />
      <LogsClient rootKeys={workspace.keys} buckets={["unkey_mutations"]} members={members} />
    </div>
  );
}
