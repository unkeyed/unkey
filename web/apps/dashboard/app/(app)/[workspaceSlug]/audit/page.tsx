import { getAuth } from "@/lib/auth";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { getWorkspace } from "./actions";
import { LogsClient } from "./components/logs-client";
export const dynamic = "force-dynamic";

export default async function AuditPage() {
  const { orgId } = await getAuth();
  const { workspace, members } = await getWorkspace(orgId);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Audit Log</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <LogsClient rootKeys={workspace.keys} buckets={["unkey_mutations"]} members={members} />
      </PageBody>
    </PageContainer>
  );
}
