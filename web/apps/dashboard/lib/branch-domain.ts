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
  const suffix = `-${environmentSlug}-${workspaceSlug}`;

  // A workspace slug can itself end in six hex characters, so try the token
  // form first and fall back to the untokened one instead of guessing.
  const tokenMatch = /-([0-9a-f]{6})$/.exec(label);
  const candidates =
    tokenMatch === null
      ? [{ label, token: "" }]
      : [
          { label: label.slice(0, -tokenMatch[0].length), token: tokenMatch[0] },
          { label, token: "" },
        ];

  for (const candidate of candidates) {
    if (!candidate.label.endsWith(suffix) || candidate.label.length === suffix.length) {
      continue;
    }
    const prefix = candidate.label.slice(0, -suffix.length);
    return `${prefix}-git-<branch>-${workspaceSlug}${candidate.token}${environmentHostname.slice(dot)}`;
  }

  return null;
}
