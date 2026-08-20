"use client";

import { getPortalByMapping, keyspaceMapping } from "@/lib/portal/client";
import {
  MAPPING_CONFLICT_MESSAGE,
  SLUG_CONFLICT_DETAIL,
  portalConflict,
} from "@/lib/portal/conflicts";
import { slugifyPortalName } from "@/lib/portal/slugify";
import { type PortalQueryResult, portalQueryKey, useCreatePortal } from "@/lib/portal/use-portal";
import { portalSlugSchema } from "@/lib/portal/validation";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import type { Portal } from "@unkey/api/models/components";
import { Button, DialogContainer, FormInput } from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({ slug: portalSlugSchema });

type FormValues = z.infer<typeof formSchema>;

type Props = {
  keyAuthId: string;
  /** Prefills the slug; the operator can overwrite it before submitting. */
  resourceName: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CreatePortalDialog({ keyAuthId, resourceName, isOpen, onOpenChange }: Props) {
  const queryClient = useQueryClient();
  // Claims every failure: this dialog stays open on error, so a toast behind it
  // would report the same thing twice, or in the mapping case say the wrong one.
  const createPortal = useCreatePortal(keyAuthId, { onError: () => true });
  const [dialogError, setDialogError] = useState<string | null>(null);
  // Spans the create *and* the conflict re-read, which `isLoading` does not.
  const [submitting, setSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    setError,
    clearErrors,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: { slug: slugifyPortalName(resourceName) },
  });

  /**
   * A 409 can also mean the create landed and the acknowledgement was lost, so
   * the row is reported as a conflict against itself. A read that finds a portal
   * settles that; a read that fails leaves the conflict to be reported.
   */
  const readKeyspacePortal = async (): Promise<Portal | null> => {
    try {
      return await getPortalByMapping(keyspaceMapping(keyAuthId));
    } catch {
      return null;
    }
  };

  const submit = async ({ slug }: FormValues) => {
    setDialogError(null);
    clearErrors("slug");
    setSubmitting(true);
    try {
      // The response carries only `{ portalId }`, so the surface re-reads the
      // full row rather than rendering a synthesized one. `useCreatePortal`
      // invalidates on success, and react-query awaits `onSuccess` before
      // `mutateAsync` resolves, so that refetch is already under way here.
      await createPortal.mutateAsync({ slug, enabled: true });
      onOpenChange(false);
      return;
    } catch (error) {
      const conflict = portalConflict(error);
      if (conflict === "mapping") {
        setDialogError(MAPPING_CONFLICT_MESSAGE);
        return;
      }
      if (conflict === "slug") {
        const existing = await readKeyspacePortal();
        if (existing) {
          // The re-read already holds the row the surface would otherwise
          // invalidate for, so seed it rather than fetching it twice.
          queryClient.setQueryData<PortalQueryResult>(portalQueryKey(keyAuthId), {
            found: true,
            portal: existing,
          });
          onOpenChange(false);
          return;
        }
        setError("slug", { type: "server", message: SLUG_CONFLICT_DETAIL });
        return;
      }
      setDialogError(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={(open) => {
        // Closing mid-flight would hide the outcome of a request already sent.
        if (!submitting) {
          onOpenChange(open);
        }
      }}
      title="Enable customer portal"
      subTitle="Choose the slug your createSession calls will use."
      footer={
        <div className="flex w-full flex-col items-center justify-center gap-2">
          <Button
            type="submit"
            form="create-portal-form"
            variant="primary"
            size="xlg"
            className="w-full"
            // Not gated on `isValid`: `handleSubmit` already blocks an invalid
            // slug, and leaving the button live means submitting shows the rule
            // rather than a dead control with no explanation.
            disabled={submitting}
            loading={submitting}
            loadingLabel="Enabling customer portal"
          >
            Enable portal
          </Button>
          <div className="text-xs text-gray-9">You can change the slug later</div>
        </div>
      }
    >
      <form id="create-portal-form" onSubmit={handleSubmit(submit)} className="flex flex-col gap-4">
        <FormInput
          label="Portal slug"
          description="Lowercase letters, numbers, and hyphens. 3-64 characters."
          placeholder="acme"
          readOnly={submitting}
          error={errors.slug?.message}
          {...register("slug")}
        />
        {dialogError ? (
          <p className="rounded-lg border border-error-6 bg-error-2 p-3 text-[13px] leading-5 text-error-11">
            {dialogError}
          </p>
        ) : null}
      </form>
    </DialogContainer>
  );
}
