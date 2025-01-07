import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { InputSearch, Ufo } from "@unkey/icons";
import { type SearchParams, getWorkspace, parseFilterParams } from "./actions";
import { Filters } from "./components/filters";
import { AuditLogTableClient } from "./components/table/audit-log-table-client";

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

            <AuditLogTableClient />
          </main>
        ) : (
          <EmptyPlaceholder>
            <EmptyPlaceholder.Icon>
              <Ufo />
            </EmptyPlaceholder.Icon>
            <EmptyPlaceholder.Title>No logs</EmptyPlaceholder.Title>
            <EmptyPlaceholder.Description>
              There are no audit logs available yet. Create a key or another resource and come back
              here.
            </EmptyPlaceholder.Description>
          </EmptyPlaceholder>
        )}
      </PageContent>
    </div>
  );
}
