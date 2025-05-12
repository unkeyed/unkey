import { LogsClient } from "@/app/(app)/apis/[apiId]/_overview/logs-client";
import { fetchApiAndWorkspaceDataFromDb } from "@/app/(app)/apis/[apiId]/actions";
import dynamic from "next/dynamic";

const ApisNavbar = dynamic(() =>
  import("@/app/(app)/apis/[apiId]/api-id-navbar").then((mod) => mod.ApisNavbar),
);

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
