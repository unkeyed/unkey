"use client";

import Link from "next/link";

export function ScheduledDeletionsHeader() {
  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
      <div className="flex flex-col gap-0.5">
        <h1 className="font-semibold text-gray-12 text-lg leading-8">Scheduled Deletions</h1>
        <p className="text-[13px] text-gray-11 leading-5">
          Resources here will be permanently deleted at the time shown. Restore them anytime before
          then to cancel. Need to delete something immediately?{" "}
          <Link
            href="mailto:support@unkey.com"
            className="text-accent-12 underline underline-offset-2 hover:text-accent-11"
          >
            Contact support
          </Link>
          .
        </p>
      </div>
    </div>
  );
}
