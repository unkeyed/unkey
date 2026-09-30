import { describe, expect, it } from "vitest";
import {
  type SourceProject,
  buildSourceTree,
  encodeSources,
  environmentIdsOf,
  selectedEnvironmentIds,
  tickedEnvironmentIds,
} from "./gateway-sources";

const tree: SourceProject[] = [
  {
    id: "project_a",
    name: "Store",
    apps: [
      {
        id: "app_api",
        name: "api",
        environments: [
          { id: "env_api_prod", name: "production" },
          { id: "env_api_preview", name: "preview" },
        ],
      },
      { id: "app_web", name: "web", environments: [{ id: "env_web_prod", name: "production" }] },
    ],
  },
  {
    id: "project_b",
    name: "Analytics",
    apps: [
      {
        id: "app_ingest",
        name: "ingest",
        environments: [{ id: "env_ingest_prod", name: "production" }],
      },
    ],
  },
];

describe("buildSourceTree", () => {
  it("hangs each environment off its own app", () => {
    const built = buildSourceTree(
      [{ id: "project_a", name: "Store", apps: [{ id: "app_api", name: "api" }] }],
      [
        { id: "env_api_prod", name: "production", appId: "app_api" },
        { id: "env_other", name: "production", appId: "app_gone" },
      ],
    );
    expect(built[0].apps[0].environments).toEqual([{ id: "env_api_prod", name: "production" }]);
  });
});

describe("selectedEnvironmentIds", () => {
  it("selects everything when no filter is set", () => {
    expect(
      selectedEnvironmentIds(tree, { projectIds: [], appIds: [], environmentIds: [] }),
    ).toEqual(environmentIdsOf(tree));
  });

  it("expands a project to its environments", () => {
    expect(
      selectedEnvironmentIds(tree, { projectIds: ["project_a"], appIds: [], environmentIds: [] }),
    ).toEqual(["env_api_prod", "env_api_preview", "env_web_prod"]);
  });

  it("intersects the three lists the way the drain query does", () => {
    expect(
      selectedEnvironmentIds(tree, {
        projectIds: ["project_a"],
        appIds: ["app_api"],
        environmentIds: ["env_api_prod", "env_ingest_prod"],
      }),
    ).toEqual(["env_api_prod"]);
  });
});

describe("encodeSources", () => {
  it("addresses whole projects by project id", () => {
    expect(
      encodeSources(tree, new Set(["env_api_prod", "env_api_preview", "env_web_prod"])),
    ).toEqual({ projectIds: ["project_a"], appIds: [], environmentIds: [] });
  });

  it("addresses whole apps by app id when projects are partial", () => {
    expect(encodeSources(tree, new Set(["env_api_prod", "env_api_preview"]))).toEqual({
      projectIds: [],
      appIds: ["app_api"],
      environmentIds: [],
    });
  });

  it("pins a mixed selection to environment ids", () => {
    expect(encodeSources(tree, new Set(["env_api_prod", "env_web_prod"]))).toEqual({
      projectIds: [],
      appIds: [],
      environmentIds: ["env_api_prod", "env_web_prod"],
    });
  });

  it("round-trips every encoding back to the same environments", () => {
    for (const selection of [
      ["env_api_prod", "env_api_preview", "env_web_prod"],
      ["env_api_prod", "env_api_preview"],
      ["env_api_prod", "env_ingest_prod"],
      ["env_web_prod"],
    ]) {
      expect(selectedEnvironmentIds(tree, encodeSources(tree, new Set(selection)))).toEqual(
        selection,
      );
    }
  });

  it("returns empty lists for an empty selection", () => {
    expect(encodeSources(tree, new Set())).toEqual({
      projectIds: [],
      appIds: [],
      environmentIds: [],
    });
  });
});

describe("tickedEnvironmentIds", () => {
  const none = { projectIds: [], appIds: [], environmentIds: [] };

  it("ticks everything in all mode", () => {
    expect(tickedEnvironmentIds(tree, "all", none)).toEqual(environmentIdsOf(tree));
  });

  it("ticks nothing when the user is picking and has picked nothing", () => {
    expect(tickedEnvironmentIds(tree, "some", none)).toEqual([]);
  });

  it("ticks the stored selection when the user is picking", () => {
    expect(tickedEnvironmentIds(tree, "some", { ...none, appIds: ["app_api"] })).toEqual([
      "env_api_prod",
      "env_api_preview",
    ]);
  });
});
