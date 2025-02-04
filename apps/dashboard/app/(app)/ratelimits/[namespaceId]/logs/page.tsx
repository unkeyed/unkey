import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navbar";
import { Badge } from "@/components/ui/badge";
import { Gauge } from "@unkey/icons";
import { navigation } from "../constants";
import { LogsClient } from "./components/logs-client";

export default async function RatelimitLogsPage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  const tenantId = getTenantId();

  const namespace = await db.query.ratelimitNamespaces.findFirst({
    where: (table, { eq, and, isNull }) => and(eq(table.id, namespaceId), isNull(table.deletedAt)),
    with: {
      workspace: true,
    },
  });
  if (!namespace || namespace.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return <LogsContainerPage namespaceId={namespaceId} namespaceName={namespace.name} />;
}

const LogsContainerPage = ({
  namespaceId,
  namespaceName,
}: {
  namespaceId: string;
  namespaceName: string;
}) => {
  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/ratelimits/${namespaceId}`} isIdentifier>
            {namespaceName.length > 0 ? namespaceName : "<Empty>"}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/ratelimits/${namespaceId}/logs`} active>
            Logs
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="namespaceId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {namespaceId}
            <CopyButton value={namespaceId} />
          </Badge>
        </Navbar.Actions>
      </Navbar>
      <SubMenu navigation={navigation(namespaceId)} segment="logs" />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
};
