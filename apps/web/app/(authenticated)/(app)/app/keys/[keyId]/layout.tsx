import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";
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
    return redirect("/onboarding");
  }
  const navigation = [
    { label: "Overview", href: `/app/keys/${props.params.keyId}`, segment: null },
    { label: "Settings", href: `/app/keys/${props.params.keyId}/settings`, segment: "settings" },
  ];

  return (
    <div>
      <PageHeader
        title={key.id}
        description="Look, it's a key"
        actions={[
          <Badge
            key="keyId"
            variant="secondary"
            className="flex justify-between w-full font-mono font-medium"
          >
            {key.id}
            <CopyButton value={key.id} className="ml-2" />
          </Badge>,
        ]}
      />
      <div className="-mt-4 md:space-x-4 ">
        <Navbar navigation={navigation} />
      </div>
      <main className="mt-8 mb-20">{props.children}</main>
    </div>
  );
}
