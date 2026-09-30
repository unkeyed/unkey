export const LIMITS_DOCS_URL = "https://www.unkey.com/docs/platform/workspaces/quotas";

// Substrings of pkg/deploy/deployfail/messages.go. Typed failure codes exist
// only on the public v2 API; the dashboard reads ctrl, which carries a bare
// string — plumbing a code through ctrl retires this table.
const LIMITS: ReadonlyArray<{ pattern: string; label: string }> = [
  { pattern: "exceeded your cpu quota", label: "CPU" },
  { pattern: "exceeded your memory quota", label: "memory" },
  { pattern: "exceeded your storage quota", label: "ephemeral disk" },
];

// No upgrade CTA on purpose: the three workspace ceilings are identical on
// every plan (lib/limits.ts), so no upgrade fixes this failure.
export function limitFailure(error: string): string | null {
  const lower = error.toLowerCase();
  const label = LIMITS.find(({ pattern }) => lower.includes(pattern))?.label;
  if (!label) {
    return null;
  }
  return `This workspace is at its ${label} limit. Stop or archive a deployment to free capacity, or request a limit raise.`;
}
