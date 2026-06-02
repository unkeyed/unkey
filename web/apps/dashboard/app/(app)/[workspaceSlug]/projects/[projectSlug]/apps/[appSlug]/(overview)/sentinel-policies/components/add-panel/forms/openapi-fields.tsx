"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { Strong } from "./summary-helpers";

export function OpenApiFields() {
  const { workspaceSlug, projectSlug } = useParams<{
    workspaceSlug: string;
    projectSlug: string;
  }>();

  return (
    <div className="flex flex-col gap-4">
      <div className="text-gray-11 text-[13px] leading-5">
        Validates incoming requests against your OpenAPI specification. Requests that don't conform
        are rejected with HTTP <Strong className="font-mono">400 Bad Request</Strong>.
      </div>

      <div className="flex items-center gap-2 rounded-md border border-grayA-4 bg-grayA-2 px-3 py-2 text-[13px] text-gray-11">
        <span className="size-2 rounded-full shrink-0 bg-success-11" />
        <span>
          {"Using auto-scraped spec. "}
          <Link
            href={`/${workspaceSlug}/projects/${projectSlug}/settings`}
            className="text-accent-12 decoration-dotted underline underline-offset-3 font-medium"
          >
            Configure scrape path
          </Link>
        </span>
      </div>
    </div>
  );
}

export function OpenApiPolicySummary() {
  return (
    <span className="text-gray-11">
      <Strong>Auto-scraped</Strong>
    </span>
  );
}
