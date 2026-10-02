import {
  priceActiveKeysMicroCents,
  priceComputeMeterMicroCents,
} from "@/lib/billing/deployPricing";
import { describe, expect, it } from "vitest";
import { breakdownFromUsage, buildComputeTree, priceUsageQuantitiesCents } from "./compute-tree";
import { buildSpendSeries } from "./spend-series";

describe("billing resource labels", () => {
  it.each([
    {
      id: "gone",
      name: null,
      labels: ["Deleted project", "Deleted app", "Deleted environment"],
      deleted: true,
    },
    {
      id: "empty",
      name: "",
      labels: ["Deleted project", "Deleted app", "Deleted environment"],
      deleted: true,
    },
    {
      id: "live",
      name: "Production",
      labels: ["Production", "Production", "Production"],
      deleted: false,
    },
    {
      id: "named-deleted",
      name: "Deleted project",
      labels: ["Deleted project", "Deleted project", "Deleted project"],
      deleted: false,
    },
    {
      id: "",
      name: null,
      labels: ["Unattributed", "Unattributed", "Unattributed"],
      deleted: false,
    },
  ])("labels $id resources without changing IDs or charges", ({ id, name, labels, deleted }) => {
    const tree = buildComputeTree({
      usage: [
        {
          projectId: id,
          projectName: name,
          appId: id,
          appName: name,
          environmentId: id,
          environmentSlug: name,
          cpuSeconds: 7200,
          memoryGiBHours: 3,
          egressGiB: 4,
          diskGiBHours: 5,
          grossMicroCents: 123,
        },
      ],
      gateway: [
        { projectId: id, projectName: name, appId: id, activeKeys: 7, grossMicroCents: 456 },
      ],
    });
    const project = tree.projects[0];
    const app = project.apps[0];
    const environment = app.environments[0];
    expect([project.name, app.name, environment.name]).toEqual(labels);
    expect([project.deleted, app.deleted, environment.deleted]).toEqual([
      deleted,
      deleted,
      deleted,
    ]);
    expect([project.projectId, app.appId, environment.environmentId]).toEqual([id, id, id]);
    expect(tree.microCents).toBe(579);
    expect(project.microCents).toBe(579);
    expect(app.microCents).toBe(123);
    expect(environment.cpuHours).toBe(2);
    expect(project.gateway).toEqual({ activeKeys: 7, microCents: 456 });
  });

  it("keeps deleted gateway-only projects and their chart series separate", () => {
    const tree = buildComputeTree({
      usage: [],
      gateway: [
        { projectId: "proj_a", projectName: null, appId: "a", activeKeys: 2, grossMicroCents: 10 },
        { projectId: "proj_b", projectName: null, appId: "b", activeKeys: 3, grossMicroCents: 20 },
      ],
    });
    expect(
      tree.projects.map(({ projectId, name, microCents }) => ({ projectId, name, microCents })),
    ).toEqual([
      { projectId: "proj_b", name: "Deleted project", microCents: 20 },
      { projectId: "proj_a", name: "Deleted project", microCents: 10 },
    ]);
    const { series, points } = buildSpendSeries({
      tree,
      rows: [
        {
          groupId: "proj_a",
          time: 0,
          cpuHours: 0,
          memoryGiBHours: 0,
          egressGiB: 2,
          diskGiBHours: 0,
        },
        {
          groupId: "proj_b",
          time: 0,
          cpuHours: 0,
          memoryGiBHours: 0,
          egressGiB: 3,
          diskGiBHours: 0,
        },
      ],
      start: 0,
      end: 1,
    });
    expect(series.map(({ key, label }) => ({ key, label }))).toEqual([
      { key: "proj_b", label: "Deleted project" },
      { key: "proj_a", label: "Deleted project" },
    ]);
    expect(points).toEqual([{ time: 0, proj_a: 10, proj_b: 15 }]);
  });
});

describe("priceUsageQuantitiesCents", () => {
  it("prices the display units with the Deploy meter rates", () => {
    expect(
      priceUsageQuantitiesCents({
        cpuHours: 1,
        memoryGiBHours: 1,
        egressGiB: 1,
        diskGiBHours: 1,
      }),
    ).toEqual({
      cpu: 2.49984,
      memory: 1.24992,
      egress: 5,
      disk: 0.0216,
    });
  });
});

describe("breakdownFromUsage", () => {
  it("prices each row and fills what the API omits", () => {
    const meters = { cpuSeconds: 7200, memoryGiBHours: 3, diskGiBHours: 5, egressGiB: 4 };
    const breakdown = breakdownFromUsage({
      environments: [
        {
          projectId: "proj_KEBAP",
          projectName: "Payments",
          appId: "app_KEBAP",
          appName: "API",
          environmentId: "env_KEBAP",
          environmentSlug: "production",
          ...meters,
        },
        { projectId: "proj_KEBAP", appId: "app_gone", environmentId: "env_gone", ...meters },
      ],
      apps: [
        {
          appId: "app_KEBAP",
          appName: "API",
          projectId: "proj_KEBAP",
          projectName: "Payments",
          activeKeys: 7,
        },
        { appId: "app_gone", activeKeys: 3 },
      ],
    });

    expect(breakdown.usage).toEqual([
      {
        projectId: "proj_KEBAP",
        projectName: "Payments",
        appId: "app_KEBAP",
        appName: "API",
        environmentId: "env_KEBAP",
        environmentSlug: "production",
        ...meters,
        grossMicroCents: priceComputeMeterMicroCents(meters),
      },
      {
        projectId: "proj_KEBAP",
        projectName: null,
        appId: "app_gone",
        appName: null,
        environmentId: "env_gone",
        environmentSlug: null,
        ...meters,
        grossMicroCents: priceComputeMeterMicroCents(meters),
      },
    ]);
    expect(breakdown.gateway).toEqual([
      {
        projectId: "proj_KEBAP",
        projectName: "Payments",
        appId: "app_KEBAP",
        activeKeys: 7,
        grossMicroCents: priceActiveKeysMicroCents(7),
      },
      {
        projectId: "",
        projectName: null,
        appId: "app_gone",
        activeKeys: 3,
        grossMicroCents: priceActiveKeysMicroCents(3),
      },
    ]);

    // The keys of a deleted app have no project, so they show as Unattributed
    const unattributed = buildComputeTree(breakdown).projects.find(
      (project) => project.projectId === "",
    );
    expect(unattributed?.name).toBe("Unattributed");
  });
});
