"use client";
import {
  METRIC_TYPE_LABELS,
  useMetricType,
} from "@/app/(app)/apis/[apiId]/_overview/hooks/use-metric-type";
import { LogsClient } from "@/app/(app)/apis/[apiId]/_overview/logs-client";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;
  const { metricType } = useMetricType();

  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/apis/${apiId}`,
          text: METRIC_TYPE_LABELS[metricType],
        }}
      />
      <LogsClient apiId={apiId} />
    </div>
  );
}
