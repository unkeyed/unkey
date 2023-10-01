import { Navbar } from "@/components/dashboard/navbar";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { PropsWithChildren } from "react";

type Props = PropsWithChildren<{
  params: {
    keyId: string;
  };
}>;
export const revalidate = 0;
export default async function ApiPageLayout(props: Props) {
  const tenantId = getTenantId();

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.id, props.params.keyId),
    with: {
      workspace: true,
    },
  });
  if (!key || key.workspace.tenantId !== tenantId) {
    return notFound();
  }
  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.keyAuthId, key.keyAuthId),
  });
  if (!api) {
    return notFound();
  }

  const navigation = [
    { label: "Overview", href: `/app/keys/${props.params.keyId}`, segment: null },
    { label: "Settings", href: `/app/keys/${props.params.keyId}/settings`, segment: "settings" },
    { label: "API", href: `/app/apis/${api.id}`, segment: api.id },
  ];

  return (
    <div>
      <Navbar navigation={navigation} />
      <main className="mt-8 mb-20">{props.children}</main>
    </div>
  );
}
