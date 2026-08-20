import type { Portal, V2PortalUpdatePortalRequestBody } from "@unkey/api/models/components";

/**
 * The editable shape of a portal, flattened for a form. Branding is spelled as
 * empty strings rather than `undefined` so an input can bind to it directly;
 * `buildPortalUpdate` translates empty back into the API's `null`.
 */
export type PortalFormValues = {
  slug: string;
  enabled: boolean;
  logoUrl: string;
  primaryColor: string;
};

/**
 * react-hook-form's record of the fields the operator actually edited. Shaped
 * to accept `formState.dirtyFields` directly.
 */
export type PortalDirtyFields = Partial<Record<keyof PortalFormValues, boolean>>;

export function portalFormValues(portal: Portal): PortalFormValues {
  return {
    slug: portal.slug,
    enabled: portal.enabled,
    logoUrl: portal.branding?.logoUrl ?? "",
    primaryColor: portal.branding?.primaryColor ?? "",
  };
}

/**
 * Builds the smallest `updatePortal` body for the fields the operator touched,
 * or `null` when they touched nothing.
 *
 * The patch is driven by `dirtyFields` rather than by diffing against the
 * portal the page currently holds. A form is seeded once at mount, but the
 * portal prop is refetched (window focus, post-mutation invalidation), so a
 * diff would read a field another operator changed in the meantime as a
 * deliberate edit and overwrite it. Only what this operator typed is sent.
 *
 * `updatePortal` is tri-state: an omitted field is unchanged, `null` clears it,
 * and a value sets it. The server rejects `""`, so a branding field the
 * operator emptied is sent as `null`.
 */
export function buildPortalUpdate(
  portalId: string,
  values: PortalFormValues,
  dirtyFields: PortalDirtyFields,
): V2PortalUpdatePortalRequestBody | null {
  // The id is unambiguous even when the slug is the field being edited.
  const body: V2PortalUpdatePortalRequestBody = { portal: portalId };
  let dirty = false;

  if (dirtyFields.slug) {
    body.slug = values.slug;
    dirty = true;
  }

  if (dirtyFields.enabled) {
    body.enabled = values.enabled;
    dirty = true;
  }

  if (dirtyFields.logoUrl) {
    body.logoUrl = values.logoUrl || null;
    dirty = true;
  }

  if (dirtyFields.primaryColor) {
    body.primaryColor = values.primaryColor || null;
    dirty = true;
  }

  return dirty ? body : null;
}
