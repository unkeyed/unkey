import { fetchApiAndWorkspaceDataFromDb } from "@/app/(app)/apis/[apiId]/actions";
import { ApisNavbar } from "@/app/(app)/apis/[apiId]/api-id-navbar";
import { KeyDetailsLogsClient } from "./logs-client";

export default async function KeyDetailsPage(props: {
  params: { apiId: string; keyAuthId: string; keyId: string };
}) {
  const apiId = props.params.apiId;
  const keyspaceId = props.params.keyAuthId;
  const keyId = props.params.keyId;

  const { currentApi, workspaceApis } = await fetchApiAndWorkspaceDataFromDb(apiId);

  return (
    <div className="min-h-screen">
      <ApisNavbar
        api={currentApi}
        activePage={{
          href: `/apis/${apiId}`,
          text: "Requests",
        }}
        apis={workspaceApis}
      />
      <KeyDetailsLogsClient keyspaceId={keyspaceId} keyId={keyId} />
    </div>
  );
}
