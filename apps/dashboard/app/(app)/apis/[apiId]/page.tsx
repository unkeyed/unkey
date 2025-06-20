"use client";
import { LogsClient } from "@/app/(app)/apis/[apiId]/_overview/logs-client";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;

  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/apis/${apiId}`,
          text: "Requests",
        }}
      />
      <LogsClient apiId={apiId} />
    </div>
  );
}
