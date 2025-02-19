import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { InputSearch } from "@unkey/icons";
import { Empty } from "@unkey/ui";
import { type SearchParams, getWorkspace, parseFilterParams } from "./actions";
import { Filters } from "./components/filters";
import { LogsClient } from "./components/logs-client";

export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  searchParams: SearchParams;
};

export default async function AuditPage(props: Props) {
  const tenantId = getTenantId();
  const workspace = await getWorkspace(tenantId);
  const parsedParams = parseFilterParams(props.searchParams);

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<InputSearch />}>
          <Navbar.Breadcrumbs.Link href="/audit">Audit</Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
      <PageContent>
        {workspace.auditLogBuckets.length > 0 ? (
          <main className="mb-5">
            <Filters
              workspace={workspace}
              parsedParams={parsedParams}
              selectedBucketName={parsedParams.bucketName}
            />

            <LogsClient />
          </main>
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
      </PageContent>
    </div>
  );
}
