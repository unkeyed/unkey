import { getTenantId } from "@/lib/auth";
import { Empty } from "@unkey/ui";
import { getWorkspace } from "./actions";
import { LogsClient } from "./components/logs-client";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function AuditPage() {
  const tenantId = getTenantId();
  const { workspace, members } = await getWorkspace(tenantId);

  return (
    <div>
      <Navigation />
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
