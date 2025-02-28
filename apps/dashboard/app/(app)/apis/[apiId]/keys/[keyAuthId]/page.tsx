import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";

import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { Nodes } from "@unkey/icons";
import { navigation } from "../../constants";
import { Keys } from "./keys";

export const dynamic = "force-dynamic";

export default async function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const orgId = await getOrgId();

  const keyAuth = await db.query.keyAuth.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAtM)),
    with: {
      workspace: true,
      api: true,
    },
  });
  if (!keyAuth || keyAuth.workspace.orgId !== orgId) {
    return notFound();
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/apis/${props.params.apiId}`} isIdentifier>
            {keyAuth.api.name}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link active href={`/apis/${props.params.apiId}/keys/${keyAuth.id}`}>
            Keys
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="apiId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {keyAuth.api.id}
            <CopyButton value={keyAuth.api.id} />
          </Badge>
          <CreateKeyButton apiId={keyAuth.api.id} keyAuthId={keyAuth.api.keyAuthId!} />
        </Navbar.Actions>
      </Navbar>

      <PageContent>
        <SubMenu navigation={navigation(keyAuth.api.id, keyAuth.id!)} segment="keys" />

        <div className="flex flex-col gap-8 mt-8 mb-20">
          <Keys keyAuthId={keyAuth.id} apiId={props.params.apiId} />
        </div>
      </PageContent>
    </div>
  );
}
