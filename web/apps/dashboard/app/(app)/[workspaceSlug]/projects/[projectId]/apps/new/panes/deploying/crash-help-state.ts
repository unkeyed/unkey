import { type RepoTreeEntry, findDockerfiles } from "../settings/deployment-config";

export type CrashHelp =
  | { type: "loading" }
  | { type: "dockerfile"; dockerfile: string; rootDirectory: string }
  | { type: "checklist"; port: number; startCommand: string; rootDirectory: string };

type CrashHelpInput = {
  settings: { dockerfile: string; dockerContext: string; port: number; command: string[] } | null;
  tree: RepoTreeEntry[] | null;
};

export function directoryLabel(path: string): string {
  return path === "." || path === "" ? "./" : path;
}

export function directoryName(path: string): string {
  return path === "." || path === "" ? "the repository root" : path;
}

export function resolveCrashHelp({ settings, tree }: CrashHelpInput): CrashHelp {
  if (!settings) {
    return { type: "loading" };
  }
  const rootDirectory = settings.dockerContext;
  const dockerfile =
    settings.dockerfile === "" && tree
      ? findDockerfiles(tree, rootDirectory)
          .toSorted((a, b) => a.split("/").length - b.split("/").length)
          .at(0)
      : undefined;
  if (dockerfile) {
    return { type: "dockerfile", dockerfile, rootDirectory };
  }
  return {
    type: "checklist",
    port: settings.port,
    startCommand: settings.command.length > 0 ? settings.command.join(" ") : "Automatic",
    rootDirectory,
  };
}
