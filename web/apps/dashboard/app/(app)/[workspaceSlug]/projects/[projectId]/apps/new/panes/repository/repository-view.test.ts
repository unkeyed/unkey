import { describe, expect, it } from "vitest";
import {
  ALL_OWNERS,
  type PickViewInput,
  type RepoItem,
  ownerOptions,
  resolveOwner,
  resolvePickView,
  visibleRepos,
} from "./repository-view";

const repo = {
  id: 1,
  fullName: "acme/storefront",
  installationId: 9,
  defaultBranch: "main",
};

const base: PickViewInput = {
  installed: true,
  contextError: null,
  installation: undefined,
  installationError: null,
  repos: [repo],
  reposError: null,
};

describe("resolvePickView", () => {
  it("waits for the installation and the list", () => {
    expect(resolvePickView({ ...base, installed: undefined })).toEqual({ kind: "loading" });
    expect(resolvePickView({ ...base, repos: undefined })).toEqual({ kind: "loading" });
  });

  it("asks to install GitHub when there is no installation", () => {
    expect(resolvePickView({ ...base, installed: false })).toEqual({ kind: "not-installed" });
  });

  it("shows a GitHub status error instead of loading forever", () => {
    expect(resolvePickView({ ...base, installed: undefined, contextError: "boom" })).toEqual({
      kind: "error",
      message: "boom",
    });
  });

  it("shows the installation error instead of loading forever", () => {
    expect(resolvePickView({ ...base, installationError: "boom" })).toEqual({
      kind: "error",
      message: "boom",
    });
  });

  it("shows the list error", () => {
    expect(resolvePickView({ ...base, reposError: "boom" })).toEqual({
      kind: "error",
      message: "boom",
    });
  });

  it("lists repositories before the app's connection loads or exists", () => {
    expect(resolvePickView(base)).toEqual({ kind: "pick", repos: [repo], linkedRepoId: null });
    expect(resolvePickView({ ...base, installation: { repoConnection: null } })).toEqual({
      kind: "pick",
      repos: [repo],
      linkedRepoId: null,
    });
  });

  it("marks the linked repository", () => {
    expect(
      resolvePickView({ ...base, installation: { repoConnection: { repositoryId: 1 } } }),
    ).toEqual({ kind: "pick", repos: [repo], linkedRepoId: 1 });
  });
});

const repos: RepoItem[] = [
  { id: 1, fullName: "acme/storefront", installationId: 9, defaultBranch: "main" },
  { id: 2, fullName: "globex/api", installationId: 7, defaultBranch: "main" },
  { id: 3, fullName: "acme/docs", installationId: 9, defaultBranch: "main" },
];

describe("repo owners", () => {
  it("offers all accounts first, then each owner once in list order", () => {
    expect(ownerOptions(repos).map((option) => option.label)).toEqual([
      "All accounts",
      "acme",
      "globex",
    ]);
  });

  it("defaults to all accounts and keeps a picked owner while it still has repos", () => {
    expect(resolveOwner(repos, null)).toBe(ALL_OWNERS);
    expect(resolveOwner(repos, "globex")).toBe("globex");
    expect(resolveOwner(repos, "initech")).toBe(ALL_OWNERS);
  });

  it("labels every repo with its owner when showing all accounts", () => {
    expect(visibleRepos(repos, ALL_OWNERS, "").map((row) => row.label)).toEqual([
      "acme/storefront",
      "globex/api",
      "acme/docs",
    ]);
  });

  it("filters by owner and drops the owner from the label", () => {
    expect(visibleRepos(repos, "acme", "").map((row) => [row.repo.id, row.label])).toEqual([
      [1, "storefront"],
      [3, "docs"],
    ]);
  });

  it("searches the owner and the repo name", () => {
    expect(visibleRepos(repos, ALL_OWNERS, " DOC ").map((row) => row.repo.id)).toEqual([3]);
    expect(visibleRepos(repos, ALL_OWNERS, "globex").map((row) => row.repo.id)).toEqual([2]);
  });
});
