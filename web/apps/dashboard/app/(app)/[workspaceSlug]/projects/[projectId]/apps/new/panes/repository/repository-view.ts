export type RepoItem = {
  id: number;
  fullName: string;
  installationId: number;
  defaultBranch: string;
};

type InstallationData = {
  repoConnection: { repositoryId: number } | null;
};

type PickView =
  | { kind: "loading" }
  | { kind: "not-installed" }
  | { kind: "error"; message: string }
  | { kind: "pick"; repos: RepoItem[]; linkedRepoId: number | null };

export type PickViewInput = {
  installed: boolean | undefined;
  contextError: string | null;
  installation: InstallationData | undefined;
  installationError: string | null;
  repos: RepoItem[] | undefined;
  reposError: string | null;
};

export function resolvePickView(input: PickViewInput): PickView {
  const loadError = input.contextError ?? input.installationError;
  if (loadError) {
    return { kind: "error", message: loadError };
  }
  if (input.installed === undefined) {
    return { kind: "loading" };
  }
  if (!input.installed) {
    return { kind: "not-installed" };
  }
  if (input.reposError) {
    return { kind: "error", message: input.reposError };
  }
  if (!input.repos) {
    return { kind: "loading" };
  }
  return {
    kind: "pick",
    repos: input.repos,
    linkedRepoId: input.installation?.repoConnection?.repositoryId ?? null,
  };
}

function repoOwner(fullName: string): string {
  return fullName.split("/")[0];
}

export function repoName(fullName: string): string {
  return fullName.slice(fullName.indexOf("/") + 1);
}

export const ALL_OWNERS = "*";

type OwnerScope = "all" | "owner";

const SCOPE_LABEL: Record<OwnerScope, (fullName: string) => string> = {
  all: (fullName) => fullName,
  owner: repoName,
};

function repoOwners(repos: RepoItem[]): string[] {
  return [...new Set(repos.map((repo) => repoOwner(repo.fullName)))];
}

export function ownerOptions(repos: RepoItem[]): { value: string; label: string }[] {
  return [
    { value: ALL_OWNERS, label: "All accounts" },
    ...repoOwners(repos).map((owner) => ({ value: owner, label: owner })),
  ];
}

export function resolveOwner(repos: RepoItem[], picked: string | null): string {
  return picked !== null && repoOwners(repos).includes(picked) ? picked : ALL_OWNERS;
}

export function visibleRepos(
  repos: RepoItem[],
  owner: string,
  query: string,
): { repo: RepoItem; label: string }[] {
  const scope: OwnerScope = owner === ALL_OWNERS ? "all" : "owner";
  const needle = query.trim().toLowerCase();
  return repos
    .filter((repo) => scope === "all" || repoOwner(repo.fullName) === owner)
    .filter((repo) => repo.fullName.toLowerCase().includes(needle))
    .map((repo) => ({ repo, label: SCOPE_LABEL[scope](repo.fullName) }));
}
