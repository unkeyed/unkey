/** Detects GitHub's `fork-owner:branch` notation so the UI can show a fork indicator before submission. */
export function parseForkRef(value: string): { forkOwner: string; branch: string } | null {
  if (value.startsWith("http://") || value.startsWith("https://")) {
    return null;
  }
  const colonIdx = value.indexOf(":");
  if (colonIdx === -1) {
    return null;
  }
  const owner = value.slice(0, colonIdx);
  const branch = value.slice(colonIdx + 1);
  if (!owner || !branch || owner.includes("/")) {
    return null;
  }
  return { forkOwner: owner, branch };
}
