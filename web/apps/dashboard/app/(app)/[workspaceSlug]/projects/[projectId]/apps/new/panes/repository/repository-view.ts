export type RepoItem = {
  id: number;
  fullName: string;
  installationId: number;
  defaultBranch: string;
};

export function repoShortName(fullName: string): string {
  return fullName.split("/").at(-1) ?? fullName;
}

export type Connection = {
  repositoryId: number;
  repositoryFullName: string;
  installationId: number;
  branch: string;
};

type InstallationData = {
  defaultBranch: string;
  repoConnection: {
    repositoryId: number;
    repositoryFullName: string;
    installationId: number;
    defaultBranch: string | null;
  } | null;
};

export type PickView =
  | { kind: "loading" }
  | { kind: "not-installed" }
  | { kind: "error"; message: string }
  | { kind: "pick"; repos: RepoItem[]; current: Connection | null };

export type SetupView =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "disconnected" }
  | { kind: "connected"; connection: Connection };

type InstallationInput = {
  installation: InstallationData | undefined;
  installationError: string | null;
};

export type PickViewInput = InstallationInput & {
  installed: boolean | undefined;
  repos: RepoItem[] | undefined;
  reposError: string | null;
};

function toConnection(installation: InstallationData): Connection | null {
  const { repoConnection } = installation;
  if (!repoConnection) {
    return null;
  }
  return {
    repositoryId: repoConnection.repositoryId,
    repositoryFullName: repoConnection.repositoryFullName,
    installationId: repoConnection.installationId,
    branch: repoConnection.defaultBranch ?? installation.defaultBranch,
  };
}

export function resolvePickView(input: PickViewInput): PickView {
  if (input.installationError) {
    return { kind: "error", message: input.installationError };
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
  const current = input.installation ? toConnection(input.installation) : null;
  return { kind: "pick", repos: input.repos, current };
}

export function resolveSetupView({
  installation,
  installationError,
  picked,
}: InstallationInput & { picked: Connection | null }): SetupView {
  if (picked) {
    return { kind: "connected", connection: picked };
  }
  if (installation === undefined) {
    return installationError ? { kind: "error", message: installationError } : { kind: "loading" };
  }
  const connection = toConnection(installation);
  return connection ? { kind: "connected", connection } : { kind: "disconnected" };
}
