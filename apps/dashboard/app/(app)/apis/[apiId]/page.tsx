import { LogsClient } from "@/app/(app)/apis/[apiId]/_overview/logs-client";
import { fetchApiAndWorkspaceDataFromDb } from "@/app/(app)/apis/[apiId]/actions";
import { ApisNavbar } from "@/app/(app)/apis/[apiId]/api-id-navbar";

export const dynamic = "force-dynamic";

export default async function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;

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
      <LogsClient apiId={apiId} />
    </div>
  );
}
