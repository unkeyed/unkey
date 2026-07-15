"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowUpRight } from "@unkey/icons";
import { Button, CopyButton, DialogContainer, FormInput, toast } from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { BrandColorField, type PortalBrandingValue } from "./portal-branding";
import { PortalPreview } from "./portal-preview";

const SAVE_DELAY_MS = 500;

const IMAGE_EXTENSION_RE = /\.(png|jpe?g|svg)$/i;

const brandingSchema = z.object({
  name: z.string().trim().min(1, "Name is required").max(50, "Name must be 50 characters or less"),
  logoUrl: z
    .string()
    .trim()
    .refine((value) => {
      if (value === "") {
        return true;
      }
      try {
        const parsed = new URL(value);
        return parsed.protocol === "https:" && IMAGE_EXTENSION_RE.test(parsed.pathname);
      } catch {
        return false;
      }
    }, "Enter a direct image URL ending in .png, .jpg, or .svg."),
  primaryColor: z.string().regex(/^#[0-9a-fA-F]{6}$/, "Use a 6-digit hex color like #18181B."),
});

// Prototype-only persistence, next to the enablement key in use-portal-lifecycle.
function brandingStorageKey(resourceId: string) {
  return `unkey:portal-branding:${resourceId}`;
}

function loadBranding(resourceId: string, fallbackName: string): PortalBrandingValue {
  const defaults: PortalBrandingValue = {
    name: fallbackName,
    logoUrl: "",
    primaryColor: "#18181B",
  };
  const raw = localStorage.getItem(brandingStorageKey(resourceId));
  if (!raw) {
    return defaults;
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null) {
      return defaults;
    }
    const record = parsed as Record<string, unknown>;
    const str = (key: keyof PortalBrandingValue) => {
      const value = record[key];
      return typeof value === "string" ? value : defaults[key];
    };
    return { name: str("name"), logoUrl: str("logoUrl"), primaryColor: str("primaryColor") };
  } catch {
    return defaults;
  }
}

// Mock save; the real implementation writes portal_branding via trpc.
async function saveBranding(resourceId: string, values: PortalBrandingValue): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, SAVE_DELAY_MS));
  localStorage.setItem(brandingStorageKey(resourceId), JSON.stringify(values));
}

export function PortalConfig({
  resourceId,
  resourceName,
  url,
  onDisable,
}: {
  resourceId: string;
  resourceName: string;
  url: string;
  onDisable: () => void;
}) {
  const [disableOpen, setDisableOpen] = useState(false);
  // Only rendered after the lifecycle hook hydrates, so localStorage is available.
  const [initialBranding] = useState<PortalBrandingValue>(() =>
    loadBranding(resourceId, resourceName),
  );
  const [saving, setSaving] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors, isValid, isDirty },
  } = useForm<z.infer<typeof brandingSchema>>({
    resolver: zodResolver(brandingSchema),
    mode: "onChange",
    defaultValues: initialBranding,
  });

  const branding = watch();

  const save = async (values: PortalBrandingValue) => {
    setSaving(true);
    await saveBranding(resourceId, values);
    reset(values);
    setSaving(false);
    toast.success("Changes saved");
  };

  return (
    <div className="flex w-full flex-col gap-6">
      <div className="w-full divide-y divide-grayA-4 overflow-hidden rounded-lg border border-grayA-4">
        <div className="flex flex-col gap-4 p-6 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-sm font-medium text-accent-12">Portal URL</h2>
          </div>
          <div className="flex min-w-0 items-center gap-1">
            <span className="truncate text-[13px] text-gray-11">{url}</span>
            <CopyButton value={url} variant="ghost" />
            <a
              href={`https://${url}`}
              target="_blank"
              rel="noopener noreferrer"
              title="Open portal"
              aria-label="Open portal"
              className="flex size-6 shrink-0 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-4 hover:text-gray-12"
            >
              <ArrowUpRight iconSize="sm-regular" />
            </a>
          </div>
        </div>
        <div className="grid gap-x-8 px-6 pt-6 lg:grid-cols-2">
          <div className="flex flex-col pb-6">
            <h2 className="text-sm font-medium text-accent-12">Branding</h2>
            <p className="mt-1 text-[13px] leading-5 text-gray-11">
              Customize how the portal looks to your users.
            </p>
            <form onSubmit={handleSubmit(save)} className="mt-6 flex flex-col gap-6">
              <FormInput
                label="Name"
                placeholder="Acme Inc."
                error={errors.name?.message}
                {...register("name")}
              />
              <FormInput
                label="Logo URL"
                description="Use a direct link to an image file ending in .png, .jpg, or .svg."
                descriptionPosition="label"
                placeholder="https://example.com/logo.png"
                error={errors.logoUrl?.message}
                {...register("logoUrl")}
              />
              <div className="flex flex-col gap-1.5">
                <span className="text-[13px] text-gray-11">Primary color</span>
                <BrandColorField
                  color={branding.primaryColor}
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
                disabled={!isDirty || !isValid || saving}
                loading={saving}
              >
                Save
              </Button>
            </form>
          </div>
          <div className="flex flex-col justify-end">
            <PortalPreview
              name={branding.name || resourceName}
              url={url}
              branding={branding}
              className="flex-1 rounded-b-none border-b-0 shadow-none"
            />
          </div>
        </div>
      </div>

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
              onClick={() => {
                setDisableOpen(false);
                onDisable();
              }}
            >
              Disable portal
            </Button>
            <div className="text-xs text-gray-9">You can re-enable it at any time</div>
          </div>
        }
      >
        <p className="text-[13px] text-gray-11">
          The portal at <span className="font-medium text-gray-12">{url}</span> stops working
          immediately and existing sessions end. Your users' API keys keep working.
        </p>
      </DialogContainer>
    </div>
  );
}
