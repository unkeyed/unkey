"use client";

import { projectSettingsPath } from "@/lib/navigation/routes/projects";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Strong } from "./summary-helpers";

export function OpenApiFields() {
  const { workspaceSlug, projectId } = useParams<{
    workspaceSlug: string;
    projectId: string;
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
            href={projectSettingsPath({ workspaceSlug, projectId })}
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
