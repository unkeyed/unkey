import { fetchApiAndWorkspaceDataFromDb, getKeyDetails } from "@/app/(app)/apis/[apiId]/actions";
import { ApisNavbar } from "@/app/(app)/apis/[apiId]/api-id-navbar";
import { KeyDetailsLogsClient } from "./logs-client";

export default async function KeyDetailsPage(props: {
  params: { apiId: string; keyAuthId: string; keyId: string };
}) {
  const apiId = props.params.apiId;
  const keyspaceId = props.params.keyAuthId;
  const keyId = props.params.keyId;

  const { currentApi, workspaceApis } = await fetchApiAndWorkspaceDataFromDb(apiId);
  const keyData = await getKeyDetails(keyId, keyspaceId, currentApi.workspaceId);

  return (
    <div className="min-h-screen">
      <ApisNavbar
        api={currentApi}
        activePage={{
          href: `/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        apis={workspaceApis}
        keyData={keyData}
      />
      <KeyDetailsLogsClient keyspaceId={keyspaceId} keyId={keyId} />
    </div>
  );
}
