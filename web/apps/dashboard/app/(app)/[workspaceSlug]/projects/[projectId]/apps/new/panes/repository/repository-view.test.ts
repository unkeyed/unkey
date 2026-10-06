import { describe, expect, it } from "vitest";
import {
  type PickViewInput,
  repoShortName,
  resolvePickView,
  resolveSetupView,
} from "./repository-view";

const repo = {
  id: 1,
  fullName: "acme/storefront",
  installationId: 9,
  defaultBranch: "main",
};

const installation = {
  defaultBranch: "main",
  repoConnection: {
    repositoryId: 1,
    repositoryFullName: "acme/storefront",
    installationId: 9,
    defaultBranch: "canary",
  },
};

const connection = {
  repositoryId: 1,
  repositoryFullName: "acme/storefront",
  installationId: 9,
  branch: "canary",
};

const base: PickViewInput = {
  installed: true,
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
    expect(resolvePickView(base)).toEqual({ kind: "pick", repos: [repo], current: null });
  });

  it("lists repositories with the connected one as current", () => {
    expect(resolvePickView({ ...base, installation })).toEqual({
      kind: "pick",
      repos: [repo],
      current: connection,
    });
  });
});

describe("resolveSetupView", () => {
  it("waits for the installation, or shows its error", () => {
    expect(
      resolveSetupView({ installation: undefined, installationError: null, picked: null }),
    ).toEqual({
      kind: "loading",
    });
    expect(
      resolveSetupView({ installation: undefined, installationError: "boom", picked: null }),
    ).toEqual({
      kind: "error",
      message: "boom",
    });
  });

  it("shows the connected repository on its selected branch", () => {
    expect(resolveSetupView({ installation, installationError: null, picked: null })).toEqual({
      kind: "connected",
      connection,
    });
  });

  it("falls back to the default branch", () => {
    const noBranch = {
      ...installation,
      repoConnection: { ...installation.repoConnection, defaultBranch: null },
    };
    const view = resolveSetupView({
      installation: noBranch,
      installationError: null,
      picked: null,
    });
    expect(view.kind === "connected" ? view.connection.branch : null).toBe("main");
  });

  it("reports an app without a repository", () => {
    expect(
      resolveSetupView({
        installation: { ...installation, repoConnection: null },
        installationError: null,
        picked: null,
      }),
    ).toEqual({ kind: "disconnected" });
  });

  it("shows the picked repository before the server has the link", () => {
    const picked = { ...connection, branch: "main" };
    expect(resolveSetupView({ installation: undefined, installationError: null, picked })).toEqual({
      kind: "connected",
      connection: picked,
    });
    expect(resolveSetupView({ installation, installationError: null, picked })).toEqual({
      kind: "connected",
      connection: picked,
    });
  });
});

describe("repoShortName", () => {
  it("drops the owner", () => {
    expect(repoShortName("acme/storefront")).toBe("storefront");
    expect(repoShortName("storefront")).toBe("storefront");
  });
});
