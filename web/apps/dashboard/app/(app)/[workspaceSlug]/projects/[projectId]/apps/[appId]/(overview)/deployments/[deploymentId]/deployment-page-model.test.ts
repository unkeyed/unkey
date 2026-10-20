import type { CustomDomain, Domain } from "@/lib/collections";
import type { Deployment } from "@/lib/collections/deploy/deployments";
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
  },
  status: "ready",
  showBuild: false,
  domains: [],
  customDomains: [],
  currentDeploymentId: null,
  ...overrides,
});

describe("deploymentPageModel", () => {
  test("picks the body from the status", () => {
    expect(deploymentPageModel(input({ status: "building" })).body).toBe("stages");
    expect(deploymentPageModel(input({ status: "failed" })).body).toBe("stages");
    expect(deploymentPageModel(input({ status: "skipped" })).body).toBe("skipped");
    expect(deploymentPageModel(input({ status: "superseded" })).body).toBe("superseded");
    expect(deploymentPageModel(input({ status: "cancelled" })).body).toBe("cancelled");
  });

  test("the build view wins over every status", () => {
    expect(deploymentPageModel(input({ status: "cancelled", showBuild: true })).body).toBe("build");
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

  test("a ready deployment with a crash-looping instance reads unhealthy", () => {
    const page = deploymentPageModel(
      input({
        deployment: {
          id: "dep-1",
          environmentId: "env-1",
          targetRegions: [],
          desiredState: "running",
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
