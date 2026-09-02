"use client";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  CopyButton,
  TimestampInfo,
} from "@unkey/ui";
import { IconChevronDownOutline12 } from "nucleo-ui-outline-12";
import { IconTriangleWarningOutline18 } from "nucleo-ui-outline-18";
import { useState } from "react";
import type { RecentError } from "./types";

export function RecentDeliveryErrors({
  entries,
  loading,
  error,
}: {
  entries?: RecentError[];
  loading: boolean;
  error: boolean;
}) {
  const [detailsOpen, setDetailsOpen] = useState(false);

  if (loading) {
    return null;
  }

  if (error) {
    return (
      <AlertBanner variant="error">
        <AlertBannerTitle>Could not load delivery error details</AlertBannerTitle>
        <AlertBannerDescription>Refresh the page to try again.</AlertBannerDescription>
      </AlertBanner>
    );
  }

  if (!entries?.length) {
    return null;
  }

  return (
    <Collapsible open={detailsOpen} onOpenChange={setDetailsOpen} className="flex flex-col gap-2">
      <AlertBanner variant="error">
        <IconTriangleWarningOutline18 className="size-4" aria-hidden="true" />
        <AlertBannerTitle>
          {entries.length === 20
            ? "At least 20 delivery attempts failed"
            : `${entries.length} delivery ${entries.length === 1 ? "attempt" : "attempts"} failed`}
        </AlertBannerTitle>
        <AlertBannerDescription>
          Review failures from the past 24 hours to find the cause. Unkey pauses the log drain when
          delivery attempts keep failing.
        </AlertBannerDescription>
        <AlertBannerActions>
          <CollapsibleTrigger
            render={<Button variant="outline" size="md" />}
            className="[&[data-panel-open]_.error-chevron]:rotate-180"
          >
            {detailsOpen ? "Hide details" : "View details"}
            <IconChevronDownOutline12
              className="error-chevron text-gray-9 transition-transform duration-200"
              aria-hidden="true"
            />
          </CollapsibleTrigger>
        </AlertBannerActions>
      </AlertBanner>
      <CollapsibleContent>
        <div className="overflow-x-auto rounded-lg border border-grayA-4 bg-background">
          <table className="w-full min-w-[680px] table-fixed border-collapse text-left">
            <colgroup>
              <col className="w-44" />
              <col className="w-32" />
              <col />
            </colgroup>
            <thead className="bg-grayA-2">
              <tr className="border-b border-grayA-4">
                <th className="px-4 py-2 text-xs font-normal text-gray-10">Time</th>
                <th className="px-4 py-2 text-xs font-normal text-gray-10">Result</th>
                <th className="px-4 py-2 text-xs font-normal text-gray-10">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-grayA-4">
              {entries.map((entry) => {
                const detail =
                  entry.responseBody ||
                  entry.error ||
                  "The destination did not return error details.";
                const status =
                  entry.responseStatus > 0
                    ? entry.responseStatus
                    : entry.outcome === "permanent_error"
                      ? "Permanent error"
                      : entry.outcome === "transient_error"
                        ? "Transient error"
                        : "Error";

                return (
                  <tr
                    key={`${entry.time}-${entry.outcome}-${entry.responseStatus}`}
                    className="align-top transition-colors hover:bg-grayA-2"
                  >
                    <td className="px-4 py-3">
                      <TimestampInfo
                        value={entry.time}
                        className="whitespace-nowrap font-mono text-gray-10 underline decoration-dotted"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <span className="inline-flex rounded-md border border-errorA-4 bg-errorA-2 px-2 py-1 font-mono text-xs font-normal text-error-11">
                        {status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-start gap-2">
                        <pre className="max-h-32 min-w-0 flex-1 overflow-auto whitespace-pre-wrap break-words font-mono text-xs leading-5 text-gray-12">
                          {formatResponseBody(detail)}
                        </pre>
                        <CopyButton
                          value={detail}
                          variant="ghost"
                          size="sm"
                          className="shrink-0"
                          aria-label={entry.responseBody ? "Copy response body" : "Copy error"}
                        />
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

function formatResponseBody(body: string): string {
  try {
    const parsed: unknown = JSON.parse(body);
    return JSON.stringify(parsed, null, 2) ?? body;
  } catch {
    return body;
  }
}
