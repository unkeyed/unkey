"use client";

import { IconEarthOutline18 } from "@unkey/icons";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { ProtoCanvas } from "../proto-canvas";
import { HelpRow, type VariantProps, num, pct } from "../shared";

export function FlowVariant(p: VariantProps) {
  const rows: Array<[string, string]> = [
    ["Key verifications", num(p.totals.verified)],
    ["Ratelimit checks", num(p.totals.rlRequests)],
    ["Blocked", pct(p.totals.rlBlockedPct)],
  ];
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>{p.title}</PageHeaderTitle>
          <PageHeaderDescription>
            Requests reach Unkey first. Unkey checks keys and limits, then sends them on to your
            apps.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody className="flex flex-col gap-6">
        <ProtoCanvas
          {...p}
          options={{
            order: "services-first",
            inbound: (
              <div className="mt-[34px] w-[200px] shrink-0 rounded-lg border border-border bg-raised shadow-xs">
                <div className="flex items-center gap-2 border-b border-border px-3 py-2 leading-5">
                  <IconEarthOutline18 className="size-4 text-gray-11" />
                  <span className="text-[13px] font-medium text-gray-12">Traffic</span>
                  <span className="ml-auto text-[11px] text-gray-9">7d</span>
                </div>
                <div className="py-1">
                  {rows.map(([k, v]) => (
                    <div key={k} className="flex items-center justify-between px-3 py-1.5 text-xs">
                      <span className="text-gray-11">{k}</span>
                      <span className="tabular-nums text-gray-12">{v}</span>
                    </div>
                  ))}
                </div>
              </div>
            ),
          }}
        />
        <HelpRow />
      </PageBody>
    </PageContainer>
  );
}
