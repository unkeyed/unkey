import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/navbar";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Gauge } from "@unkey/icons";
import { redirect } from "next/navigation";
import { LogsClient } from "./_overview/logs-client";
import type { Interval } from "./filters";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RatelimitNamespacePage(props: {
  params: { namespaceId: string };
  searchParams: {
    interval?: Interval;
    identifier?: string;
  };
}) {
  const tenantId = getTenantId();

  const namespace = await db.query.ratelimitNamespaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.namespaceId), isNull(table.deletedAt)),
    with: {
      workspace: {
        columns: {
          tenantId: true,
        },
      },
    },
  });

  if (!namespace || namespace.workspace.tenantId !== tenantId) {
    return redirect("/ratelimits");
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/ratelimits/${props.params.namespaceId}`}
            isIdentifier
            active
          >
            {namespace.name.length > 0 ? namespace.name : "<Empty>"}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="namespaceId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {props.params.namespaceId}
            <CopyButton value={props.params.namespaceId} />
          </Badge>
        </Navbar.Actions>
      </Navbar>
      {/* <SubMenu */}
      {/*   navigation={navigation(props.params.namespaceId)} */}
      {/*   segment="overview" */}
      {/* /> */}
      <LogsClient namespaceId={namespace.id} />
    </div>
  );
}
