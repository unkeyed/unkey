"use client";
import { Empty } from "@unkey/ui";

export function SentinelPoliciesError() {
  return (
    <div className="border border-errorA-4 bg-errorA-2 rounded-[14px] overflow-hidden">
      <div className="flex items-center justify-center py-16 px-4">
        <Empty className="w-100 flex items-start">
          <Empty.Title className="text-error-11">Failed to load policies</Empty.Title>
          <Empty.Description className="text-left">
            Something went wrong while loading sentinel policies. Try refreshing the page.
          </Empty.Description>
        </Empty>
      </div>
    </div>
  );
}
