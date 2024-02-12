import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { PropsWithChildren } from "react";

type Props = PropsWithChildren<{
  params: {
    apiId: string;
  };
}>;

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPageLayout(props: Props) {
  const tenantId = await getTenantId();

  const api = await db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.apiId), isNull(table.deletedAt)),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return notFound();
  }
  const navigation = [
    {
      label: "Overview",
      href: `/app/apis/${props.params.apiId}`,
      segment: null,
    },
    {
      label: "Keys",
      href: `/app/apis/${props.params.apiId}/keys`,
      segment: "keys",
    },
    {
      label: "Settings",
      href: `/app/apis/${props.params.apiId}/settings`,
      segment: "settings",
    },
  ];

  return (
    <div>
      <PageHeader
        title={api.name}
        description="Manage your API"
        actions={[
          <Badge
            key="apiId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {api.id}
            <CopyButton value={api.id} />
          </Badge>,
          <CreateKeyButton apiId={api.id} />,
        ]}
      />

      <Navbar navigation={navigation} className="z-20" />

      <main className="relative mt-8 mb-20 ">{props.children}</main>
    </div>
  );
}
