import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { InputSearch } from "@unkey/icons";
import { type SearchParams, getWorkspace, parseFilterParams } from "./actions";
import { Filters } from "./components/filters";
import { AuditLogTableClient } from "./components/table/audit-log-table-client";

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

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<InputSearch />}>
          <Navbar.Breadcrumbs.Link href="/audit/unkey_mutations">Audit</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/audit/${props.params.bucket}`} active isIdentifier>
            {workspace.ratelimitNamespaces.find((ratelimit) => ratelimit.id === props.params.bucket)
              ?.name ?? props.params.bucket}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
      <PageContent>
        <main className="mb-5">
          <Filters workspace={workspace} parsedParams={parsedParams} bucket={parsedParams.bucket} />
          <AuditLogTableClient />
        </main>
      </PageContent>
    </div>
  );
}
