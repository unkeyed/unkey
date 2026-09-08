"use client";

import { getPortalByKeyspace } from "@/lib/portal/client";
import {
  AMBIGUOUS_CONFLICT_DETAIL,
  CONFLICT_UNRESOLVED_MESSAGE,
  MAPPING_CONFLICT_MESSAGE,
  SLUG_CONFLICT_DETAIL,
  portalConflict,
} from "@/lib/portal/conflicts";
import { slugifyPortalName } from "@/lib/portal/slugify";
import { type PortalQueryResult, portalQueryKey, useCreatePortal } from "@/lib/portal/use-portal";
import { portalDisplayNameSchema, portalSlugSchema } from "@/lib/portal/validation";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import type { Portal } from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { Button, DialogContainer, FormInput } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({ displayName: portalDisplayNameSchema, slug: portalSlugSchema });

type FormValues = z.infer<typeof formSchema>;

type Props = {
  keyAuthId: string;
  /** Prefills both name fields. */
  resourceName: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CreatePortalDialog({ keyAuthId, resourceName, isOpen, onOpenChange }: Props) {
  const queryClient = useQueryClient();
  // Claims every failure: the dialog stays open on error and renders it inline,
  // so a toast behind it would duplicate or contradict that.
  const createPortal = useCreatePortal(keyAuthId, { onError: () => true });
  const [dialogError, setDialogError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    setError,
    clearErrors,
    setValue,
    watch,
    formState: { errors, dirtyFields, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: { displayName: resourceName, slug: slugifyPortalName(resourceName) },
  });

  const displayName = watch("displayName");
  // The slug tracks the display name only until the operator edits it directly.
  const slugFollowsDisplayName = !dirtyFields.slug;
  useEffect(() => {
    if (slugFollowsDisplayName) {
      setValue("slug", slugifyPortalName(displayName), { shouldValidate: true });
    }
  }, [displayName, slugFollowsDisplayName, setValue]);

  // `absent` and `unknown` are separate outcomes: only a 404 proves this
  // keyspace has no portal, and collapsing a failed read into "no portal" would
  // tell an operator whose create actually landed to go pick another slug.
  type KeyspaceRead =
    | { status: "found"; portal: Portal }
    | { status: "absent" }
    | { status: "unknown" };

  const readKeyspacePortal = async (): Promise<KeyspaceRead> => {
    try {
      return { status: "found", portal: await getPortalByKeyspace(keyAuthId) };
    } catch (error) {
      if (error instanceof NotFoundErrorResponse) {
        return { status: "absent" };
      }
      return { status: "unknown" };
    }
  };

  const adopt = (portal: Portal) => {
    queryClient.setQueryData<PortalQueryResult>(portalQueryKey(keyAuthId), {
      found: true,
      portal,
    });
    onOpenChange(false);
  };

  const submit = async ({ slug, displayName: submittedName }: FormValues) => {
    setDialogError(null);
    clearErrors("slug");
    try {
      // The response carries only `{ portalId }`, so the surface re-reads the
      // full row; `useCreatePortal` invalidates on success.
      await createPortal.mutateAsync({ slug, displayName: submittedName, enabled: true });
      onOpenChange(false);
      return;
    } catch (error) {
      const conflict = portalConflict(error);
      if (conflict === null) {
        setDialogError(getErrorMessage(error));
        return;
      }

      // Any duplicate arm can be a lost ack, so all three re-read to find out
      // whether the create actually landed.
      const read = await readKeyspacePortal();

      if (read.status === "unknown" && conflict !== "mapping") {
        setDialogError(CONFLICT_UNRESOLVED_MESSAGE);
        return;
      }

      if (conflict === "mapping") {
        // A readable portal means this workspace owns it, whatever its slug.
        // The support message is only for a holder this operator cannot see.
        if (read.status === "found") {
          adopt(read.portal);
          return;
        }
        setDialogError(MAPPING_CONFLICT_MESSAGE);
        return;
      }

      // Only this submit's own row if the portal on this keyspace carries the
      // submitted slug. Anything else is a genuine collision.
      if (read.status === "found" && read.portal.slug === slug) {
        adopt(read.portal);
        return;
      }
      setError("slug", {
        type: "server",
        message: conflict === "slug" ? SLUG_CONFLICT_DETAIL : AMBIGUOUS_CONFLICT_DETAIL,
      });
    }
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={(open) => {
        // Closing mid-flight would hide the outcome of a request already sent.
        if (!isSubmitting) {
          onOpenChange(open);
        }
      }}
      title="Enable customer portal"
      subTitle="Name the portal your customers will use."
      footer={
        <div className="flex w-full flex-col items-center justify-center gap-2">
          <Button
            type="submit"
            form="create-portal-form"
            variant="primary"
            size="xlg"
            className="w-full"
            disabled={isSubmitting}
            loading={isSubmitting}
            loadingLabel="Enabling customer portal"
          >
            Enable portal
          </Button>
          <div className="text-xs text-gray-9">You can change both later</div>
        </div>
      }
    >
      <form id="create-portal-form" onSubmit={handleSubmit(submit)} className="flex flex-col gap-4">
        <FormInput
          label="Display name"
          description="Shown to your users in the portal header."
          placeholder="Acme"
          readOnly={isSubmitting}
          error={errors.displayName?.message}
          {...register("displayName")}
        />
        <FormInput
          label="Portal slug"
          description="Used in API calls, never shown to your users. Lowercase letters, numbers, and hyphens."
          placeholder="acme"
          readOnly={isSubmitting}
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
