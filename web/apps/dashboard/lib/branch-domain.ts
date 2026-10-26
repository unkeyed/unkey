// Reverses the per-environment and per-branch hostnames built by buildDomains
// in svc/ctrl/worker/deploy/domains.go. Change both together.
//
// Every generated label ends with the app's token, six hex characters derived
// from the app ID. Strip it first, then rebuild the branch label from what is
// left, so the pattern keeps the same token.
//
// A route created before the token existed ends with the workspace slug
// instead. Those still parse, without a token.
export function branchDomainPattern(
  environmentHostname: string,
  environmentSlug: string,
  workspaceSlug: string,
): string | null {
  const dot = environmentHostname.indexOf(".");
  if (dot === -1) {
    return null;
  }
  const label = environmentHostname.slice(0, dot);
  const tokenMatch = /-([0-9a-f]{6})$/.exec(label);
  const token = tokenMatch === null ? "" : tokenMatch[0];
  const withoutToken = token === "" ? label : label.slice(0, -token.length);
  const suffix = `-${environmentSlug}-${workspaceSlug}`;
  if (!withoutToken.endsWith(suffix) || withoutToken.length === suffix.length) {
    return null;
  }
  const prefix = withoutToken.slice(0, -suffix.length);
  return `${prefix}-git-<branch>-${workspaceSlug}${token}${environmentHostname.slice(dot)}`;
}
