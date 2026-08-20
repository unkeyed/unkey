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

export function portalFormValues(portal: Portal): PortalFormValues {
  return {
    slug: portal.slug,
    enabled: portal.enabled,
    logoUrl: portal.branding?.logoUrl ?? "",
    primaryColor: portal.branding?.primaryColor ?? "",
  };
}

/**
 * Diffs a portal against its edited form values and returns the smallest
 * `updatePortal` body that expresses the change, or `null` when nothing did.
 *
 * `updatePortal` is tri-state: an omitted field is unchanged, `null` clears it,
 * and a value sets it. The server rejects `""`, so a branding field the operator
 * emptied has to be sent as `null` while one that was never set and is still
 * empty has to be left out entirely. Keeping that mapping here rather than in a
 * submit handler is what makes it testable.
 */
export function buildPortalUpdate(
  original: Portal,
  modified: PortalFormValues,
): V2PortalUpdatePortalRequestBody | null {
  const previous = portalFormValues(original);
  // The id is unambiguous even when the slug is the field being edited.
  const body: V2PortalUpdatePortalRequestBody = { portal: original.id };
  let dirty = false;

  if (modified.slug !== previous.slug) {
    body.slug = modified.slug;
    dirty = true;
  }

  if (modified.enabled !== previous.enabled) {
    body.enabled = modified.enabled;
    dirty = true;
  }

  if (modified.logoUrl !== previous.logoUrl) {
    body.logoUrl = modified.logoUrl || null;
    dirty = true;
  }

  if (modified.primaryColor !== previous.primaryColor) {
    body.primaryColor = modified.primaryColor || null;
    dirty = true;
  }

  return dirty ? body : null;
}
