// Reverses the per-environment and per-branch hostnames built by buildDomains
// in svc/ctrl/worker/deploy/domains.go. Change both together.
//
// Every generated label ends with the app code, six hex characters derived from
// the app ID. Strip it first, then rebuild the branch label from what is left,
// so the pattern keeps the same code.
//
// A route created before the code existed ends with the workspace slug
// instead. Those still parse, without a code.
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

  // A workspace slug can itself end in six hex characters, so try the form
  // with a code first and fall back to the one without instead of guessing.
  const codeMatch = /-([0-9a-f]{6})$/.exec(label);
  const candidates =
    codeMatch === null
      ? [{ label, code: "" }]
      : [
          { label: label.slice(0, -codeMatch[0].length), code: codeMatch[0] },
          { label, code: "" },
        ];

  for (const candidate of candidates) {
    if (!candidate.label.endsWith(suffix) || candidate.label.length === suffix.length) {
      continue;
    }
    const prefix = candidate.label.slice(0, -suffix.length);
    return `${prefix}-git-<branch>-${workspaceSlug}${candidate.code}${environmentHostname.slice(dot)}`;
  }

  return null;
}
