"use client";

import { Label } from "@/components/ui/label";
import type { OIDCVercelInput } from "@/lib/trpc/routers/oidc-vercel/validation";
import { Input } from "@unkey/ui";
import { cn } from "cn";
import type { ComponentProps } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";

// Adapted from https://ui.shadcn.com/oidc/r/provider-vercel-rhf-zod.json.
// Unkey's fixed trust policy uses names, requires an environment, and generates its audience.
type Fields = Pick<OIDCVercelInput, "vercelTeamSlug" | "vercelProjectName" | "vercelEnvironment">;

function ClaimChip({ className, ...props }: ComponentProps<"code">) {
  return (
    <code
      data-slot="claim-chip"
      className={cn("ml-auto font-mono text-xs font-normal text-gray-10", className)}
      {...props}
    />
  );
}

export function OIDCVercelPolicyFields({ disabled }: { disabled: boolean }) {
  const form = useFormContext<Fields>();
  const teamSlug = useWatch({ control: form.control, name: "vercelTeamSlug" });

  return (
    <fieldset disabled={disabled} data-slot="vercel-policy-form" className="flex flex-col gap-4">
      <legend className="mb-4 text-sm font-medium text-gray-12">Vercel trust policy</legend>
      <Controller
        control={form.control}
        name="vercelTeamSlug"
        render={({ field, fieldState }) => (
          <div data-invalid={fieldState.invalid} className="flex flex-col gap-2">
            <Label htmlFor="vc-teamSlug" className="flex w-full items-center text-gray-12">
              Team slug <ClaimChip>iss / owner</ClaimChip>
            </Label>
            <Input
              {...field}
              id="vc-teamSlug"
              placeholder="acme"
              required
              aria-invalid={fieldState.invalid}
              aria-describedby="vc-teamSlug-description vc-teamSlug-error"
              onChange={(event) => field.onChange(event.target.value.trim())}
            />
            <p id="vc-teamSlug-description" className="text-xs text-gray-10">
              The team URL segment: vercel.com/acme.
            </p>
            {fieldState.invalid && (
              <p id="vc-teamSlug-error" role="alert" className="text-xs text-error-11">
                {fieldState.error?.message}
              </p>
            )}
          </div>
        )}
      />
      <Controller
        control={form.control}
        name="vercelProjectName"
        render={({ field, fieldState }) => (
          <div data-invalid={fieldState.invalid} className="flex flex-col gap-2">
            <Label htmlFor="vc-project" className="flex w-full items-center text-gray-12">
              Project name <ClaimChip>project</ClaimChip>
            </Label>
            <Input
              {...field}
              id="vc-project"
              placeholder="storefront"
              required
              aria-invalid={fieldState.invalid}
              aria-describedby="vc-project-description vc-project-error"
              onChange={(event) => field.onChange(event.target.value.trim())}
            />
            <p id="vc-project-description" className="text-xs text-gray-10">
              The project value in the Vercel OIDC subject.
            </p>
            {fieldState.invalid && (
              <p id="vc-project-error" role="alert" className="text-xs text-error-11">
                {fieldState.error?.message}
              </p>
            )}
          </div>
        )}
      />
      <Controller
        control={form.control}
        name="vercelEnvironment"
        render={({ field, fieldState }) => (
          <div data-invalid={fieldState.invalid} className="flex flex-col gap-2">
            <Label htmlFor="vc-environment" className="flex w-full items-center text-gray-12">
              Environment <ClaimChip>environment</ClaimChip>
            </Label>
            <Input
              {...field}
              id="vc-environment"
              placeholder="production"
              required
              aria-invalid={fieldState.invalid}
              aria-describedby="vc-environment-description vc-environment-error"
              onChange={(event) => field.onChange(event.target.value.trim())}
            />
            <p id="vc-environment-description" className="text-xs text-gray-10">
              Use development, preview, production, or a Custom Environment slug.
            </p>
            {fieldState.invalid && (
              <p id="vc-environment-error" role="alert" className="text-xs text-error-11">
                {fieldState.error?.message}
              </p>
            )}
          </div>
        )}
      />
      {teamSlug && (
        <p className="break-all text-xs text-gray-10">
          Team issuer: <code className="font-mono">https://oidc.vercel.com/{teamSlug}</code>
        </p>
      )}
    </fieldset>
  );
}
