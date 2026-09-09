"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { ProtoPicker } from "./components/proto-picker";
import { useMetricsUrlState } from "./hooks/use-app-metrics";
import { TimelineVariant } from "./variants/timeline";
import { TotalsVariant } from "./variants/totals";

const VARIANTS = ["Timeline", "Totals"];

export default function MetricsPage() {
  const state = useMetricsUrlState();

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Metrics</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        {state.environmentId ? (
          state.variant === 2 ? (
            <TotalsVariant key="totals" state={{ ...state, environmentId: state.environmentId }} />
          ) : (
            <TimelineVariant
              key="timeline"
              state={{ ...state, environmentId: state.environmentId }}
            />
          )
        ) : (
          <p className="text-[13px] text-gray-10">
            {state.isEnvironmentsLoading
              ? "Loading environments…"
              : "This app has no environments."}
          </p>
        )}
      </PageBody>
      <ProtoPicker
        variants={VARIANTS}
        active={state.variant - 1}
        onChange={(i) => state.setVariant(i + 1)}
      />
    </PageContainer>
  );
}
