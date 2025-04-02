import { PageContent } from "@/components/page-content";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { ApisNavbar } from "../../api-id-navbar";
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
      <ApisNavbar
        api={keyAuth.api}
        activePage={{
          href: `/apis/${keyAuth.api}/Keys`,
          text: "Keys",
        }}
        apis={[keyAuth.api]}
      />
      <PageContent>
        <div className="flex flex-col gap-8 mt-8 mb-20">
          <Keys keyAuthId={keyAuth.id} apiId={props.params.apiId} />
        </div>
      </PageContent>
    </div>
  );
}
