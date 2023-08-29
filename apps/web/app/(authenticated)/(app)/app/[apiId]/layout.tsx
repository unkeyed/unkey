import { NavLink } from "@/components/dashboard/api-navbar";
import { CopyButton } from "@/components/dashboard/copy-button";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import Link from "next/link";
import { redirect } from "next/navigation";
import { PropsWithChildren } from "react";
import { DeleteApiButton } from "./delete-api-button";

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
    return redirect("/onboarding");
  }
  const navigation: { name: string; href: string }[] = [
    { name: "Overview", href: `/app/${props.params.apiId}` },
    { name: "Keys", href: `/app/${props.params.apiId}/keys` },
  ];

  return (
    <div>
      <PageHeader
        title={api.name}
        description={"Here is a list of your current API keys"}
        actions={[
          <Badge
            key="apiId"
            variant="secondary"
            className="flex justify-between w-full font-mono font-medium"
          >
            {api.id}
            <CopyButton value={api.id} className="ml-2" />
          </Badge>,
          <Link key="new" href={`/app/${api.id}/keys/new`}>
            <Button variant="secondary">Create Key</Button>
          </Link>,
          <DeleteApiButton key="delete-api" apiId={api.id} apiName={api.name} />,
        ]}
      />
      <div className="-mt-4 md:space-x-4 ">
        {navigation.map((item) => (
          <NavLink key={item.name} href={item.href} label={item.name} />
        ))}
      </div>
      <Separator className="mb-6" />
      {props.children}
    </div>
  );
}
