import { fetchApiAndWorkspaceDataFromDb } from "../../actions";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

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
          href: `/apis/${apiId}/keys_v2/${keyspaceId}`,
          text: "Keys",
        }}
        apis={workspaceApis}
      />
      <KeysClient apiId={props.params.apiId} keyspaceId={keyspaceId} />
    </div>
  );
}
