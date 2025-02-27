import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";

import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navigation/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { Nodes } from "@unkey/icons";
import { navigation } from "../../constants";
import { Keys } from "./keys";
import { Navigation } from "./navigation";

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
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAtM)),
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
      <Navigation apiId={props.params.apiId} keyA={keyAuth}/>
      

      <PageContent>
        <SubMenu navigation={navigation(keyAuth.api.id, keyAuth.id!)} segment="keys" />

        <div className="flex flex-col gap-8 mt-8 mb-20">
          <Keys keyAuthId={keyAuth.id} apiId={props.params.apiId} />
        </div>
      </PageContent>
    </div>
  );
}
