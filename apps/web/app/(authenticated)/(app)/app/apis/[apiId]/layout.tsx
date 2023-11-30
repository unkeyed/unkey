import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import Link from "next/link";
import { redirect } from "next/navigation";
import { PropsWithChildren } from "react";

type Props = PropsWithChildren<{
  params: {
    apiId: string;
  };
}>;
export const revalidate = 0;
export default async function ApiPageLayout(props: Props) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/new");
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
        description={" "}
        actions={[
          <Badge
            key="apiId"
            variant="secondary"
            className="flex justify-between w-full font-mono font-medium"
          >
            {api.id}
            <CopyButton value={api.id} className="ml-2" />
          </Badge>,
          <Link key="new" href={`/app/apis/${api.id}/keys/new`}>
            <Button variant="secondary">Create Key</Button>
          </Link>,
        ]}
      />
      <div className="-mt-4 md:space-x-4 ">
        <Navbar navigation={navigation} />
      </div>
      <main className="mt-8 mb-20">{props.children}</main>
    </div>
  );
}
