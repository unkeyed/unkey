import { getAuth } from "@/lib/auth";
import { dashboardAuditLogBuckets } from "@unkey/schema/src/auditlog";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { getWorkspace } from "./actions";
import { LogsClient } from "./components/logs-client";
export const dynamic = "force-dynamic";

export default async function AuditPage() {
  const { orgId, role } = await getAuth();
  const { workspace } = await getWorkspace(orgId, role);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Audit Log</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <LogsClient rootKeys={workspace.keys} buckets={[...dashboardAuditLogBuckets]} />
      </PageBody>
    </PageContainer>
  );
}
