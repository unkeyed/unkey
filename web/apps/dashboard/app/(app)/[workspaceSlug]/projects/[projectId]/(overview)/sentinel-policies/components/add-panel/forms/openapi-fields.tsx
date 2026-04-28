"use client";

import { FormTextarea } from "@unkey/ui";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { PolicyFormValues } from "../schema";
import { Strong } from "./summary-helpers";

type OpenapiFormValues = Extract<PolicyFormValues, { type: "openapi" }>;

export function OpenApiFields() {
  const { control } = useFormContext<OpenapiFormValues>();
  const specYaml = useWatch({ control, name: "specYaml" });
  const hasCustomSpec = (specYaml ?? "").trim().length > 0;
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

      <Controller
        control={control}
        name="specYaml"
        render={({ field, fieldState }) => (
          <FormTextarea
            label="OpenAPI Spec"
            rows={15}
            requirement="optional"
            descriptionPosition="label"
            placeholder={"openapi: '3.0.0'\ninfo:\n  title: My API\n  version: '1.0'"}
            description="Supports OpenAPI 3.0, 3.1, and 3.2 in YAML or JSON. Leave empty to use the spec scraped from your deployment."
            className="text-xs min-h-[160px]"
            value={field.value ?? ""}
            onChange={field.onChange}
            error={fieldState.error?.message}
          />
        )}
      />

      {!hasCustomSpec && (
        <div className="flex items-center gap-2 rounded-md border border-grayA-4 bg-grayA-2 px-3 py-2 text-[13px] text-gray-11">
          <span className="size-2 rounded-full shrink-0 bg-success-11" />
          <span>
            {"Using auto-scraped spec. "}
            <Link
              href={`/${workspaceSlug}/projects/${projectId}/settings`}
              className="text-accent-12 decoration-dotted underline underline-offset-3 font-medium"
            >
              Configure scrape path
            </Link>
          </span>
        </div>
      )}
    </div>
  );
}

export function OpenApiPolicySummary() {
  const { control } = useFormContext<OpenapiFormValues>();
  const specYaml = useWatch({ control, name: "specYaml" });

  return (
    <span className="text-gray-11">
      {specYaml ? <Strong>Custom spec</Strong> : <Strong>Auto-scraped</Strong>}
    </span>
  );
}
