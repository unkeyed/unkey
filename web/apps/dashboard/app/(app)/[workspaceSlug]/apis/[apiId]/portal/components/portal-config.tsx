"use client";

import { buildPortalUpdate, portalFormValues } from "@/lib/portal/build-update";
import { SLUG_CONFLICT_DETAIL, portalConflict } from "@/lib/portal/conflicts";
import { useUpdatePortal } from "@/lib/portal/use-portal";
import {
  logoUrlSchema,
  portalDisplayNameSchema,
  portalSlugSchema,
  primaryColorSchema,
} from "@/lib/portal/validation";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Portal } from "@unkey/api/models/components";
import {
  Button,
  CopyButton,
  DialogContainer,
  FormField,
  FormInput,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  SettingsDangerZone,
  toast,
} from "@unkey/ui";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { DeletePortalRow } from "./delete-portal-row";
import { BrandColorField } from "./portal-branding";
import { PortalPreview } from "./portal-preview";

// The preview renders the logo URL in an `<img>`, so a live value would issue
// one request per keystroke against the dashboard's own origin.
const LOGO_PREVIEW_DEBOUNCE_MS = 300;

const formSchema = z.object({
  slug: portalSlugSchema,
  displayName: portalDisplayNameSchema,
  logoUrl: logoUrlSchema,
  primaryColor: primaryColorSchema,
});

type FormValues = z.infer<typeof formSchema>;

// Only a slug conflict can reach this surface: `updatePortal` never changes a
// portal's mapping, so its mapping-availability check returns early.
function isSlugConflict(error: unknown): boolean {
  return portalConflict(error) === "slug";
}

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
};

export function PortalConfig({ portal, keyAuthId }: Props) {
  const [disableOpen, setDisableOpen] = useState(false);

  // Claim the slug conflict so it lands on the field instead of in a toast.
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
    const body = buildPortalUpdate(
      portal.id,
      { ...submitted, enabled: portal.enabled },
      dirtyFields,
    );
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
        <div className="grid gap-x-8 px-6 pt-6 lg:grid-cols-2">
          <div className="flex flex-col pb-6">
            <h2 className="text-sm font-medium text-accent-12">Branding</h2>
            <p className="mt-1 text-[13px] leading-5 text-gray-11">
              Customize how the portal looks to your users.
            </p>
            <form onSubmit={handleSubmit(save)} className="mt-6 flex flex-col gap-6">
              <FormInput
                label="Display name"
                description="Shown to your users in the portal header."
                descriptionPosition="label"
                placeholder="Acme"
                error={errors.displayName?.message}
                {...register("displayName")}
              />
              <div className="flex flex-col gap-1.5">
                <FormField
                  label="Slug"
                  description="Lowercase letters, numbers, and hyphens. 3-64 characters."
                  descriptionPosition="label"
                  error={errors.slug?.message}
                >
                  {(field) => (
                    <InputGroup variant={field.variant}>
                      <InputGroupInput
                        id={field.id}
                        placeholder="acme"
                        aria-describedby={field.describedBy}
                        aria-invalid={field.invalid}
                        {...register("slug")}
                      />
                      {/* The saved slug, not the draft: an unsaved slug is a value
                          `createSession` rejects. */}
                      <InputGroupAddon align="inline-end">
                        <CopyButton value={portal.slug} variant="ghost" />
                      </InputGroupAddon>
                    </InputGroup>
                  )}
                </FormField>
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
              displayName={values.displayName || portal.displayName}
              branding={{ logoUrl: previewLogoUrl, primaryColor: values.primaryColor }}
              className="flex-1 rounded-b-none border-b-0 shadow-none"
            />
          </div>
        </div>
      </div>

      {portal.enabled ? (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-grayA-4 p-4">
          <div className="space-y-1">
            <p className="text-sm font-normal text-gray-12">Disable portal</p>
            <p className="text-[13px] text-gray-11">
              By disabling this users will lose access to the portal immediately. Their keys will
              keep working.
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
              onClick={async () => {
                try {
                  await disablePortal.mutateAsync({ portal: portal.id, enabled: false });
                  setDisableOpen(false);
                } catch {
                  // The hook surfaced the failure; keep the dialog open to retry.
                }
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
