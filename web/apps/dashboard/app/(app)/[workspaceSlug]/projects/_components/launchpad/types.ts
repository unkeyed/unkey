import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import type { Route } from "next";

export type LaunchpadKind = "keyspace" | "ratelimit";

export type LaunchpadRow = {
  id: string;
  name: string;
  kind: LaunchpadKind;
  keyCount: number;
  total: number;
  failed: number;
  buckets: { ok: number; bad: number }[];
  projectName: string | null;
  href: Route;
};

export type RecentDeployRow = {
  id: string;
  appName: string;
  projectName: string;
  environment: string;
  status: DeploymentStatus;
  branch: string;
  commitMessage: string | null;
  authorHandle: string | null;
  authorAvatarUrl: string | null;
  prNumber: number | null;
  createdAt: number;
  href: Route;
};

export type LaunchpadModel = {
  isLoading: boolean;
  rows: LaunchpadRow[];
  keyspaces: LaunchpadRow[];
  ratelimits: LaunchpadRow[];
  identityCount: number;
  /** Newest deployment per branch and environment, newest first. */
  recentDeploys: RecentDeployRow[];
  readyDeploys: RecentDeployRow[];
  previewDeploys: RecentDeployRow[];
  identitiesHref: Route;
  keyspacesHref: Route;
  ratelimitsHref: Route;
  projectCount: number;
  windowHours: number;
  /** A workspace with nothing to jump to gets the get-started face instead. */
  isEmpty: boolean;
};

export type SparkMode = "off" | "bars" | "hover";
export type Density = "compact" | "default" | "roomy";

export type LaunchpadOptions = {
  spark: SparkMode;
  density: Density;
  showProject: boolean;
  forceEmpty: boolean;
};

export type VariantProps = {
  model: LaunchpadModel;
  options: LaunchpadOptions;
};
