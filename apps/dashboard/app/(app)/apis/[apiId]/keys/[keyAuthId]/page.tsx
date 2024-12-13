import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";

import { Keys } from "./keys";
import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { Navbar } from "@/components/navbar";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageContent } from "@/components/page-content";
import { Nodes } from "@unkey/icons";
import { navigation } from "../../constants";
import { Badge } from "@/components/ui/badge";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
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

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/apis/${props.params.apiId}`}
            isIdentifier
          >
            {keyAuth.api.name}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            active
            href={`/apis/${props.params.apiId}/keys/${keyAuth.id}`}
          >
            Keys
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <div className="flex items-center gap-2">
            <Badge
              key="apiId"
              variant="secondary"
              className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
            >
              {keyAuth.api.id}
              <CopyButton value={keyAuth.api.id} />
            </Badge>
            <CreateKeyButton
              apiId={keyAuth.api.id}
              keyAuthId={keyAuth.api.keyAuthId!}
            />
          </div>
        </Navbar.Actions>
      </Navbar>

      <PageContent>
        <SubMenu
          navigation={navigation(keyAuth.api.id, keyAuth.id!)}
          segment="keys"
        />

        <div className="flex flex-col gap-8 mt-8 mb-20">
          <Keys keyAuthId={keyAuth.id} apiId={props.params.apiId} />
        </div>
      </PageContent>
    </div>
  );
}
