type GitRefResult = { kind: "pr"; value: number } | { kind: "ref"; value: string };

/** Normalizes user input (branch name, commit SHA, or GitHub URL including /pull/, /tree/, /commit/) into a typed result so the mutation knows whether to resolve a PR or pass a ref directly to the backend. */
export function parseGitRef(raw: string): GitRefResult {
  const trimmed = raw.trim();

  const prMatch = trimmed.match(/^https?:\/\/github\.com\/[^/]+\/[^/]+\/pull\/(\d+)\/?$/);
  if (prMatch) {
    return { kind: "pr", value: Number.parseInt(prMatch[1], 10) };
  }

  const treeMatch = trimmed.match(/^https?:\/\/github\.com\/[^/]+\/[^/]+\/tree\/(.+)$/);
  if (treeMatch) {
    return { kind: "ref", value: treeMatch[1] };
  }

  const commitMatch = trimmed.match(
    /^https?:\/\/github\.com\/[^/]+\/[^/]+\/commit\/([0-9a-f]{40})$/i,
  );
  if (commitMatch) {
    return { kind: "ref", value: commitMatch[1] };
  }

  return { kind: "ref", value: trimmed };
}
