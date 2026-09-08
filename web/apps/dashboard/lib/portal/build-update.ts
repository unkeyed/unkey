import type { Portal, V2PortalUpdatePortalRequestBody } from "@unkey/api/models/components";

// Branding is spelled as empty strings so an input can bind to it directly.
// `buildPortalUpdate` translates empty back into the API's `null`.
export type PortalFormValues = {
  slug: string;
  displayName: string;
  enabled: boolean;
  logoUrl: string;
  primaryColor: string;
};

export type PortalDirtyFields = Partial<Record<keyof PortalFormValues, boolean>>;

// Flattens a portal into the form's shape.
export function portalFormValues(portal: Portal): PortalFormValues {
  return {
    slug: portal.slug,
    displayName: portal.displayName,
    enabled: portal.enabled,
    logoUrl: portal.branding?.logoUrl ?? "",
    primaryColor: portal.branding?.primaryColor ?? "",
  };
}

/**
 * Builds the smallest `updatePortal` body for the fields the operator touched,
 * or `null` when they touched nothing.
 *
 * Driven by `dirtyFields`, not by diffing the portal the page holds: that prop
 * refetches under the mounted form, so a diff would ship a field the operator
 * never touched.
 *
 * `updatePortal` is tri-state: omitted is unchanged, `null` clears, a value
 * sets. The server rejects `""`, so an emptied branding field is sent as `null`.
 */
export function buildPortalUpdate(
  portalId: string,
  values: PortalFormValues,
  dirtyFields: PortalDirtyFields,
): V2PortalUpdatePortalRequestBody | null {
  const body: V2PortalUpdatePortalRequestBody = { portal: portalId };
  let dirty = false;

  if (dirtyFields.slug) {
    body.slug = values.slug;
    dirty = true;
  }

  if (dirtyFields.displayName) {
    body.displayName = values.displayName;
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
