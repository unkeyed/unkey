import { LogsClient } from "./_overview/logs-client";
import { fetchApiAndWorkspaceDataFromDb } from "./actions";
import { ApisNavbar } from "./api-id-navbar";
export const dynamic = "force-dynamic";

export default async function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;

  const { currentApi, workspaceApis } = await fetchApiAndWorkspaceDataFromDb(apiId);

  return (
    <div className="min-h-screen max-md:pb-6">
      <ApisNavbar
        api={currentApi}
        activePage={{
          href: `/apis/${apiId}`,
          text: "Requests",
        }}
        apis={workspaceApis}
      />
      <LogsClient apiId={apiId} />
    </div>
  );
}
