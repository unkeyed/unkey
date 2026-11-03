import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import type { CustomDomain, Deployment, Domain } from "@/lib/deploy-data";
import { describe, expect, test } from "vitest";
import {
  type DeploymentPageInput,
  deploymentPageModel,
  resolvePageStatus,
} from "./deployment-page-model";

const domain = (id: string, fullyQualifiedDomainName: string): Domain => ({
  id,
  fullyQualifiedDomainName,
  projectId: "proj-1",
  appId: "app-1",
  deploymentId: "dep-1",
  environmentId: "env-1",
  sticky: "none",
  createdAt: 0,
  updatedAt: null,
});

const customDomain: CustomDomain = {
  id: "cd-1",
  domain: "api.acme.dev",
  projectId: "proj-1",
  appId: "app-1",
  environmentId: "env-1",
  verificationStatus: "verified",
  dnsRecords: [],
  verificationError: null,
  domainConnectProvider: null,
  domainConnectUrl: null,
  createdAt: 0,
  updatedAt: null,
};

const waiting = (
  statusReason: string,
): NonNullable<Deployment["instances"][number]["lastExit"]> => ({
  restartCount: 0,
  exitCode: null,
  signal: null,
  reason: null,
  finishedAt: null,
  statusReason,
});

const input = (overrides: Partial<DeploymentPageInput> = {}): DeploymentPageInput => ({
  deployment: {
    id: "dep-1",
    environmentId: "env-1",
    instances: [],
    targetRegions: [],
    desiredState: "running",
    source: "git",
  },
  status: "ready",
  buildRequested: false,
  domains: [],
  domainsLoading: false,
  customDomains: [],
  currentDeploymentId: null,
  ...overrides,
});

describe("deploymentPageModel", () => {
  test("shows the stages for every status except skipped", () => {
    const body = (status: DeploymentStatus) => deploymentPageModel(input({ status })).body;
    expect(body("building")).toBe("stages");
    expect(body("failed")).toBe("stages");
    expect(body("superseded")).toBe("stages");
    expect(body("cancelled")).toBe("stages");
    expect(body("skipped")).toBe("skipped");
  });

  test("?build asks for the build stage and keeps the status body", () => {
    const model = deploymentPageModel(input({ status: "building", buildRequested: true }));
    expect(model.body).toBe("stages");
    expect(model.requestedStage).toBe("build");
    expect(deploymentPageModel(input({ status: "building" })).requestedStage).toBe(null);
  });

  test("a prebuilt image or a skipped deployment has no build stage to request", () => {
    const prebuilt = deploymentPageModel(
      input({
        deployment: { ...input().deployment, source: "oci" },
        status: "cancelled",
        buildRequested: true,
      }),
    );
    expect(prebuilt.requestedStage).toBe(null);
    expect(prebuilt.body).toBe("stages");
    expect(
      deploymentPageModel(input({ status: "skipped", buildRequested: true })).requestedStage,
    ).toBe(null);
  });

  test("a current deployment lists its verified custom domain first", () => {
    const { domains } = deploymentPageModel(
      input({
        domains: [domain("d-1", "acme-abc.unkey.app"), domain("d-2", "api.acme.dev")],
        customDomains: [customDomain],
        currentDeploymentId: "dep-1",
      }),
    );
    expect(domains.all.map((d) => d.hostname)).toEqual(["api.acme.dev", "acme-abc.unkey.app"]);
    expect(domains.primary?.hostname).toBe("api.acme.dev");
  });

  test("counts domains once they load", () => {
    const domains = [domain("d-1", "acme-abc.unkey.app"), domain("d-2", "api.acme.dev")];
    expect(deploymentPageModel(input({ domains })).domainCount).toBe(2);
    expect(deploymentPageModel(input({ domains, domainsLoading: true })).domainCount).toBe(null);
  });

  test("a ready deployment with a crash-looping instance reads unhealthy", () => {
    const page = deploymentPageModel(
      input({
        deployment: {
          ...input().deployment,
          instances: [
            {
              id: "i-1",
              region: { id: "r-1", name: "us-east-1", platform: "aws" },
              status: "running",
              lastExit: waiting("CrashLoopBackOff"),
            },
          ],
        },
      }),
    );
    expect(page.instances).toEqual({ state: "unhealthy", running: 0, total: 1 });
    expect(page.pageStatus.label).toBe("Unhealthy");
  });
});

describe("resolvePageStatus", () => {
  test("a ready deployment with unhealthy instances reads Unhealthy", () => {
    const status = resolvePageStatus("ready", { state: "unhealthy", running: 0, total: 1 });
    expect(status.label).toBe("Unhealthy");
    expect(status.dotClass).toBe("bg-error-9");
  });

  test("a ready deployment with running instances reads Ready", () => {
    expect(resolvePageStatus("ready", { state: "running", running: 1, total: 1 }).label).toBe(
      "Ready",
    );
  });

  test("building statuses are in flight", () => {
    expect(resolvePageStatus("building", null).inFlight).toBe(true);
  });
});
