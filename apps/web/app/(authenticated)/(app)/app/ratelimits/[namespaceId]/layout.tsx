import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { PropsWithChildren } from "react";

type Props = PropsWithChildren<{
  params: {
    namespaceId: string;
  };
}>;

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RatelimitNamespacePageLayout(props: Props) {
  const tenantId = getTenantId();

  const namespace = await db.query.ratelimitNamespaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.namespaceId), isNull(table.deletedAt)),
    with: {
      workspace: true,
    },
  });
  if (!namespace || namespace.workspace.tenantId !== tenantId) {
    return notFound();
  }
  const navigation = [
    {
      label: "Overview",
      href: `/app/ratelimits/${namespace.id}`,
      segment: null,
    },

    {
      label: "Settings",
      href: `/app/ratelimits/${namespace.id}/settings`,
      segment: "settings",
    },
  ];

  return (
    <div>
      <PageHeader
        title={namespace.name}
        description="Manage your ratelimit namespace"
        actions={[
          <Badge
            key="namespaceId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {namespace.id}
            <CopyButton value={namespace.id} />
          </Badge>,
        ]}
      />

      <Navbar navigation={navigation} className="z-20" />

      <main className="relative mt-8 mb-20 ">{props.children}</main>
    </div>
  );
}
