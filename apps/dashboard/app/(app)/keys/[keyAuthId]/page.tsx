import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { Keys } from "./keys";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPage(props: { params: { apiId: string; keyAuthId: string } }) {
  const tenantId = getTenantId();

  const keyAuth = await db.query.keyAuth.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAt)),
    with: {
      workspace: true,
      api: true,
    },
  });
  if (!keyAuth || keyAuth.workspace.tenantId !== tenantId) {
    return notFound();
  }

  const navigation = [
    {
      label: "Overview",
      href: `/keys/${keyAuth.id}`,
      segment: null,
    },
    {
      label: "API",
      href: `/apis/${keyAuth.api.id}`,
      segment: "settings",
    },
  ];

  return (
    <>
      <PageHeader
        title={keyAuth.api?.name ?? keyAuth.id}
        description="Manage your KeySpace"
        actions={[
          <Badge
            key="key_auth_id"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {keyAuth.id}
            <CopyButton value={keyAuth.id} />
          </Badge>,
          <CreateKeyButton keyAuthId={keyAuth.id} />,
        ]}
      />
      <Navbar navigation={navigation} />
      <div className="flex flex-col gap-8 mt-8 mb-20">
        <Keys keyAuthId={keyAuth.id} />
      </div>
    </>
  );
}
