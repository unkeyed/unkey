import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { omitLastItemForPagination } from "@/lib/trpc/routers/audit/fetch";
import { InputSearch } from "@unkey/icons";
import { Box } from "lucide-react";
import {
  getAuditLogsForBucket,
  getWorkspace,
  parseFilterParams,
  SearchParams,
} from "./actions";
import { Filters } from "./components/filters";
import { AuditLogTableServer } from "./components/table/audit-log-table-server";

export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  params: {
    bucket: string;
  };
  searchParams: SearchParams;
};

export default async function AuditPage(props: Props) {
  const tenantId = getTenantId();
  const workspace = await getWorkspace(tenantId);
  const parsedParams = parseFilterParams({
    ...props.searchParams,
    bucket: props.params.bucket,
  });
  const bucketLogs = await getAuditLogsForBucket(workspace, parsedParams);

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<InputSearch />}>
          <Navbar.Breadcrumbs.Link href="/audit/unkey_mutations">
            Audit
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/audit/${props.params.bucket}`}
            active
            isIdentifier
          >
            {workspace.ratelimitNamespaces.find(
              (ratelimit) => ratelimit.id === props.params.bucket
            )?.name ?? props.params.bucket}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
      <PageContent>
        <main className="mb-5">
          <Filters
            workspace={workspace}
            parsedParams={parsedParams}
            bucket={parsedParams.bucket}
          />
          {!bucketLogs ? (
            <EmptyPlaceholder>
              <EmptyPlaceholder.Icon>
                <Box />
              </EmptyPlaceholder.Icon>
              <EmptyPlaceholder.Title>Bucket Not Found</EmptyPlaceholder.Title>
              <EmptyPlaceholder.Description>
                The specified audit log bucket does not exist or you do not have
                access to it.
              </EmptyPlaceholder.Description>
            </EmptyPlaceholder>
          ) : (
            <AuditLogTableServer
              logs={omitLastItemForPagination(bucketLogs.logs).slicedItems}
              selectedEvents={parsedParams.selectedEvents}
              selectedUsers={parsedParams.selectedUsers}
              selectedRootKeys={parsedParams.selectedRootKeys}
            />
          )}
        </main>
      </PageContent>
    </div>
  );
}
