import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { PropsWithChildren } from "react";
type Props = PropsWithChildren<{
  params: {
    keyId: string;
  };
}>;

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApiPageLayout(props: Props) {
  const tenantId = getTenantId();

  const key = await db.query.keys.findFirst({
    where: and(eq(schema.keys.id, props.params.keyId), isNull(schema.keys.deletedAt)),
    with: {
      workspace: true,
    },
  });
  if (!key || key.workspace.tenantId !== tenantId) {
    return notFound();
  }
  const api = await db.query.apis.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.keyAuthId, key.keyAuthId), isNull(table.deletedAt)),
  });
  if (!api) {
    return notFound();
  }

  const navigation = [
    {
      label: "Overview",
      href: `/app/keys/${props.params.keyId}`,
      segment: null,
    },
    {
      label: "Settings",
      href: `/app/keys/${props.params.keyId}/settings`,
      segment: "settings",
    },
    { label: "API", href: `/app/apis/${api.id}`, segment: api.id },
  ];

  return (
    <div>
      <PageHeader
        title={props.params.keyId}
        description="Here is an overview of your key usage"
        actions={[
          <Badge
            key="keyId"
            variant="secondary"
            className="ph-no-capture flex w-full justify-between font-mono font-medium gap-2"
          >
            {key.id}
            <CopyButton value={key.id} />
          </Badge>,
        ]}
      />
      <Navbar navigation={navigation} />
      <main className="mb-20 mt-8">{props.children}</main>
    </div>
  );
}
