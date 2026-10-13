// Reverses the per-environment and per-branch hostnames built by buildDomains
// in svc/ctrl/worker/deploy/domains.go. Change both together.
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
  const suffix = `-${environmentSlug}-${workspaceSlug}`;
  if (!label.endsWith(suffix) || label.length === suffix.length) {
    return null;
  }
  const prefix = label.slice(0, -suffix.length);
  return `${prefix}-git-<branch>-${workspaceSlug}${environmentHostname.slice(dot)}`;
}
