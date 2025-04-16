import { fetchApiAndWorkspaceDataFromDb } from "../../actions";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";
import { PageContent } from "@/components/page-content";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { Keys } from "./keys";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";

export default async function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const apiId = props.params.apiId;
  const keyspaceId = props.params.keyAuthId;

  const { currentApi, workspaceApis } = await fetchApiAndWorkspaceDataFromDb(apiId);

  return (
    <div>
      <ApisNavbar
        api={currentApi}
        activePage={{
          href: `/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        apis={workspaceApis}
      />
      <KeysClient apiId={props.params.apiId} keyspaceId={keyspaceId} />
      <Navigation apiId={props.params.apiId} keyAuth={keyAuth} />
      <PageContent>
        <div className="flex flex-col gap-8 mt-8 mb-20">
          <Keys keyAuthId={keyAuth.id} apiId={props.params.apiId} />
        </div>
      </PageContent>
    </div>
  );
}
