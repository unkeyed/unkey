"use client";

import { buildPortalUpdate, portalFormValues } from "@/lib/portal/build-update";
import { SLUG_CONFLICT_DETAIL, portalConflict } from "@/lib/portal/conflicts";
import { useDeletePortal, useUpdatePortal } from "@/lib/portal/use-portal";
import { logoUrlSchema, portalSlugSchema, primaryColorSchema } from "@/lib/portal/validation";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Portal } from "@unkey/api/models/components";
import { TriangleWarning2 } from "@unkey/icons";
import {
  Button,
  CopyButton,
  DialogContainer,
  FormInput,
  Input,
  SettingsDangerZone,
  SettingsZoneRow,
  toast,
} from "@unkey/ui";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { BrandColorField } from "./portal-branding";
import { PortalPreview } from "./portal-preview";

/**
 * The preview renders the logo URL in an `<img>`, so an unvalidated live value
 * would issue one request per keystroke, most of them relative prefixes hitting
 * the dashboard's own origin.
 */
const LOGO_PREVIEW_DEBOUNCE_MS = 300;

const formSchema = z.object({
  slug: portalSlugSchema,
  logoUrl: logoUrlSchema,
  primaryColor: primaryColorSchema,
});

type FormValues = z.infer<typeof formSchema>;

/**
 * Only a slug conflict can reach this surface: `updatePortal` never changes a
 * portal's mapping, and its handler returns early from the mapping-availability
 * check when the request carries no mapping.
 */
function isSlugConflict(error: unknown): boolean {
  return portalConflict(error) === "slug";
}

/**
 * Holds back the value the preview renders until it parses as a logo URL and
 * the operator has stopped typing.
 */
function useDebouncedLogoUrl(value: string, initial: string): string {
  const [debounced, setDebounced] = useState(initial);

  useEffect(() => {
    const next = logoUrlSchema.safeParse(value).success ? value : "";
    const timer = setTimeout(() => setDebounced(next), LOGO_PREVIEW_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [value]);

  return debounced;
}

type Props = {
  portal: Portal;
  keyAuthId: string;
  /** The mapped API's name. Read-only: a portal carries no name of its own. */
  resourceName: string;
};

export function PortalConfig({ portal, keyAuthId, resourceName }: Props) {
  const [disableOpen, setDisableOpen] = useState(false);

  // The slug conflict is claimed so it lands on the field instead of in a
  // toast; every other failure keeps the hook's default toast.
  const updatePortal = useUpdatePortal(keyAuthId, { onError: isSlugConflict });
  const disablePortal = useUpdatePortal(keyAuthId);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    setError,
    clearErrors,
    reset,
    formState: { errors, isValid, isDirty, isSubmitting, dirtyFields },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: portalFormValues(portal),
  });

  const values = watch();
  const previewLogoUrl = useDebouncedLogoUrl(values.logoUrl, portal.branding?.logoUrl ?? "");

  const save = async (submitted: FormValues) => {
    clearErrors("slug");
    // `enabled` is not on this form; disabling is its own action below.
    const body = buildPortalUpdate(portal, { ...submitted, enabled: portal.enabled });
    if (!body) {
      return;
    }
    try {
      await updatePortal.mutateAsync(body);
      reset(submitted);
      toast.success("Changes saved");
    } catch (error) {
      if (isSlugConflict(error)) {
        setError("slug", { type: "server", message: SLUG_CONFLICT_DETAIL });
        return;
      }
      // Anything else already reached the operator as a toast.
    }
  };

  return (
    <div className="flex w-full flex-col gap-6">
      <div className="w-full divide-y divide-grayA-4 overflow-hidden rounded-lg border border-grayA-4">
        <div className="flex flex-col gap-4 p-6 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-sm font-medium text-accent-12">Portal slug</h2>
            <p className="mt-1 text-[13px] leading-5 text-gray-11">
              Pass this as <span className="font-mono text-gray-12">portal</span> when you call{" "}
              <span className="font-mono text-gray-12">createSession</span>. Serves keys for{" "}
              <span className="font-medium text-gray-12">{resourceName}</span>.
            </p>
          </div>
          <div className="flex min-w-0 items-center gap-1">
            {/* The saved slug, never the form draft: an unsaved slug is a value
                `createSession` rejects. */}
            <span className="truncate font-mono text-[13px] text-gray-11">{portal.slug}</span>
            <CopyButton value={portal.slug} variant="ghost" />
          </div>
        </div>
        <div className="grid gap-x-8 px-6 pt-6 lg:grid-cols-2">
          <div className="flex flex-col pb-6">
            <h2 className="text-sm font-medium text-accent-12">Branding</h2>
            <p className="mt-1 text-[13px] leading-5 text-gray-11">
              Customize how the portal looks to your users.
            </p>
            <form onSubmit={handleSubmit(save)} className="mt-6 flex flex-col gap-6">
              <div className="flex flex-col gap-1.5">
                <FormInput
                  label="Slug"
                  description="Lowercase letters, numbers, and hyphens. 3-64 characters."
                  descriptionPosition="label"
                  placeholder="acme"
                  error={errors.slug?.message}
                  {...register("slug")}
                />
                {dirtyFields.slug ? (
                  <p className="text-[13px] leading-5 text-warning-11">
                    Changing the slug breaks every <span className="font-mono">createSession</span>{" "}
                    call that still passes the old one. Live sessions keep working.
                  </p>
                ) : null}
              </div>
              <FormInput
                label="Logo URL"
                description="A direct https:// link to your logo image. Leave empty for none."
                descriptionPosition="label"
                placeholder="https://example.com/logo.png"
                error={errors.logoUrl?.message}
                {...register("logoUrl")}
              />
              <div className="flex flex-col gap-1.5">
                <span className="text-[13px] text-gray-11">Primary color</span>
                <BrandColorField
                  color={values.primaryColor}
                  onChange={(primaryColor) =>
                    setValue("primaryColor", primaryColor, {
                      shouldValidate: true,
                      shouldDirty: true,
                    })
                  }
                />
                {errors.primaryColor?.message && (
                  <span className="text-[13px] leading-5 text-error-11">
                    {errors.primaryColor.message}
                  </span>
                )}
              </div>
              <Button
                type="submit"
                variant="primary"
                size="md"
                className="self-start"
                disabled={!isDirty || !isValid || isSubmitting}
                loading={isSubmitting}
              >
                Save
              </Button>
            </form>
          </div>
          <div className="flex flex-col justify-end">
            <PortalPreview
              slug={values.slug || portal.slug}
              branding={{ logoUrl: previewLogoUrl, primaryColor: values.primaryColor }}
              className="flex-1 rounded-b-none border-b-0 shadow-none"
            />
          </div>
        </div>
      </div>

      {portal.enabled ? (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-grayA-4 p-4">
          <div className="space-y-1">
            <p className="text-sm font-medium text-gray-12">Disable portal</p>
            <p className="text-[13px] text-gray-11">
              Your users lose access to the portal immediately. Their keys keep working.
            </p>
          </div>
          <Button variant="outline" color="danger" onClick={() => setDisableOpen(true)}>
            Disable portal
          </Button>
        </div>
      ) : null}

      <SettingsDangerZone>
        <DeletePortalRow portal={portal} keyAuthId={keyAuthId} />
      </SettingsDangerZone>

      <DialogContainer
        isOpen={disableOpen}
        onOpenChange={setDisableOpen}
        title="Disable customer portal?"
        footer={
          <div className="flex w-full flex-col items-center justify-center gap-2">
            <Button
              type="button"
              variant="primary"
              color="danger"
              size="xlg"
              className="w-full"
              loading={disablePortal.isLoading}
              loadingLabel="Disabling customer portal"
              onClick={() => {
                setDisableOpen(false);
                disablePortal.mutate({ portal: portal.id, enabled: false });
              }}
            >
              Disable portal
            </Button>
            <div className="text-xs text-gray-9">You can re-enable it at any time</div>
          </div>
        }
      >
        <p className="text-[13px] text-gray-11">
          The portal <span className="font-medium text-gray-12">{portal.slug}</span> stops working
          immediately and existing sessions end. Your users' API keys keep working.
        </p>
      </DialogContainer>
    </div>
  );
}

/**
 * Deleting is not disabling: the row is gone, every live session is revoked by
 * `RevokePortalSessionsByPortal`, and the slug returns to the pool. The
 * type-to-confirm string is the slug because a portal carries no name.
 */
function DeletePortalRow({ portal, keyAuthId }: { portal: Portal; keyAuthId: string }) {
  const [isOpen, setIsOpen] = useState(false);
  const deletePortal = useDeletePortal(keyAuthId);

  // No resolver: the only gate is the exact-match comparison below, so
  // validating the field as well would run a schema nothing reads.
  const {
    register,
    watch,
    handleSubmit,
    formState: { isSubmitting },
  } = useForm<{ confirmation: string }>({
    defaultValues: { confirmation: "" },
  });

  const confirmed = watch("confirmation") === portal.slug;

  const onSubmit = async () => {
    try {
      await deletePortal.mutateAsync({ portal: portal.id });
      setIsOpen(false);
      toast.success("Customer portal deleted");
    } catch {
      // The hook already surfaced the failure; the dialog stays open so the
      // operator can retry without retyping the slug.
    }
  };

  return (
    <>
      <SettingsZoneRow
        title="Delete this portal"
        description="Permanently remove the portal, end every live end-user session, and free the slug for reuse."
        action={{
          label: "Delete portal",
          onClick: () => setIsOpen(true),
        }}
      />

      <DialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Delete customer portal"
        subTitle="Permanently remove this portal and end every live session"
        footer={
          <div className="flex w-full flex-col items-center justify-center gap-2">
            <Button
              type="submit"
              form="delete-portal-form"
              variant="primary"
              color="danger"
              size="xlg"
              className="w-full rounded-lg"
              disabled={!confirmed || isSubmitting}
              loading={isSubmitting}
              loadingLabel="Deleting customer portal"
            >
              Delete portal
            </Button>
            <div className="text-xs text-gray-9">
              This action cannot be undone – proceed with caution
            </div>
          </div>
        }
      >
        <div className="flex items-center gap-4 rounded-xl border border-errorA-3 bg-errorA-2 px-[22px] py-6 dark:bg-black">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-error-9">
            <TriangleWarning2 iconSize="sm-regular" className="text-white" />
          </div>
          <div className="text-[13px] leading-6 text-error-12">
            <span className="font-medium">Warning:</span> deleting{" "}
            <span className="font-medium">{portal.slug}</span> is permanent. Every live end-user
            session ends immediately, and the slug becomes available for reuse. Your users' API keys
            keep working.
          </div>
        </div>
        <form id="delete-portal-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="mt-4 flex flex-col gap-1">
            <p className="text-[13px] text-gray-11">
              Type <span className="font-medium text-gray-12">{portal.slug}</span> to confirm
            </p>
            <Input
              aria-label="Portal slug confirmation"
              placeholder={`Enter "${portal.slug}" to confirm`}
              {...register("confirmation")}
            />
          </div>
        </form>
      </DialogContainer>
    </>
  );
}
