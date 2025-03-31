import { Navigation } from "@/components/navigation/navigation";
import { getOrgId } from "@/lib/auth";
import { InputSearch } from "@unkey/icons";
import { getWorkspace } from "./actions";
import { LogsClient } from "./components/logs-client";
export const dynamic = "force-dynamic";

export default async function AuditPage() {
  const orgId = await getOrgId();
  const { workspace, members } = await getWorkspace(orgId);

  return (
    <div>
      <Navigation href="/audit" name="Audit" icon={<InputSearch />} />
      <LogsClient rootKeys={workspace.keys} buckets={["unkey_mutations"]} members={members} />
    </div>
  );
}
