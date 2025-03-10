import { Navigation } from "@/components/navigation/navigation";
import { getTenantId } from "@/lib/auth";
import { InputSearch } from "@unkey/icons";
import { Empty } from "@unkey/ui";
import { getWorkspace } from "./actions";
import { LogsClient } from "./components/logs-client";
export const dynamic = "force-dynamic";

export default async function AuditPage() {
  const tenantId = await getTenantId();
  const { workspace, members } = await getWorkspace(tenantId);

  return (
    <div>
      <Navigation href="/audit" name="Audit" icon={<InputSearch />} />
      {workspace.auditLogBuckets.length > 0 ? (
        <LogsClient
          rootKeys={workspace.keys}
          buckets={workspace.auditLogBuckets}
          members={members}
        />
      ) : (
        <Empty>
          <Empty.Icon />
          <Empty.Title>No logs</Empty.Title>
          <Empty.Description>
            There are no audit logs available yet. Create a key or another resource and come back
            here.
          </Empty.Description>
        </Empty>
      )}
    </div>
  );
}
