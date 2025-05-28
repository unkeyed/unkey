import dynamic from "next/dynamic";
import { fetchApiAndWorkspaceDataFromDb } from "../../actions";
import { KeysClient } from "./_components/keys-client";

const ApisNavbar = dynamic(() =>
  import("@/app/(app)/apis/[apiId]/api-id-navbar").then((mod) => mod.ApisNavbar),
);

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
    </div>
  );
}
