import type { EnvironmentKind } from "@/lib/queries/environments";
import type { Badge } from "@unkey/ui";
import type { ComponentProps } from "react";

export const ENGINES = ["postgres", "mysql"] as const;
export const CLUSTERS = ["single", "ha"] as const;
export const REGIONS = [
  "us-east-2",
  "us-east-1",
  "us-west-2",
  "eu-central-1",
  "eu-west-2",
  "ap-southeast-1",
] as const;

export type Engine = (typeof ENGINES)[number];
export type Cluster = (typeof CLUSTERS)[number];
export type RegionId = (typeof REGIONS)[number];
type SizeId = "PS-5" | "PS-10" | "PS-20" | "PS-40";

export type Compute = { cluster: Cluster; size: SizeId };

type EnvironmentBranch = { branch: string; compute: Compute | null };

export type EnvironmentBranches = Record<EnvironmentKind, EnvironmentBranch>;

export type DatabaseLink = {
  id: string;
  name: string;
  engine: Engine;
  region: RegionId;
  createdAt: number;
  planetscale: { org: string; branches: EnvironmentBranches };
};

export type MetricPoint = [unixSeconds: number, value: number];

export type DatabaseVitals = {
  queries: MetricPoint[];
  latencyP95Ms: number;
  cpuRatio: number;
  connections: number;
};

export type DatabaseInstance = Pick<
  PlanetscaleDatabase,
  "tableCount" | "schemaUpdatedAt" | "lastBackupAt" | "maintenanceAt" | "vitals"
> & { branchCount: number };

export type Database = DatabaseLink &
  ({ status: "provisioning" } | { status: "ready"; instance: DatabaseInstance });

export type DatabaseStatus = Database["status"];

export type CreateDatabaseInput =
  | {
      source: "new";
      org: string;
      name: string;
      engine: Engine;
      region: RegionId;
      compute: Compute;
      branches: EnvironmentBranches;
    }
  | { source: "existing"; org: string; name: string; branches: EnvironmentBranches };

export const engineCopy: Record<Engine, { label: string; title: string; description: string }> = {
  postgres: {
    label: "Postgres by PlanetScale",
    title: "Postgres",
    description: "PlanetScale Postgres 18",
  },
  mysql: {
    label: "MySQL by PlanetScale",
    title: "MySQL",
    description: "PlanetScale Vitess, MySQL 8",
  },
};

export const clusterCopy: Record<Cluster, { label: string; description: string }> = {
  single: {
    label: "Single node",
    description:
      "One primary node. If it fails, the database is down until it restarts. Good for previews and side projects.",
  },
  ha: {
    label: "High availability",
    description:
      "One primary and two replicas in other zones. If the primary fails, a replica takes over. 99.99% SLA.",
  },
};

export const regionLabel: Record<RegionId, string> = {
  "us-east-1": "us-east-1 (N. Virginia)",
  "us-east-2": "us-east-2 (Ohio)",
  "us-west-2": "us-west-2 (Oregon)",
  "eu-central-1": "eu-central-1 (Frankfurt)",
  "eu-west-2": "eu-west-2 (London)",
  "ap-southeast-1": "ap-southeast-1 (Singapore)",
};

export const sizeSpec: Record<SizeId, { cpu: string; memory: string }> = {
  "PS-5": { cpu: "1/16 vCPU", memory: "512 MB" },
  "PS-10": { cpu: "1/8 vCPU", memory: "1 GB" },
  "PS-20": { cpu: "1/4 vCPU", memory: "2 GB" },
  "PS-40": { cpu: "1/2 vCPU", memory: "4 GB" },
};

export function sharesProductionBranch(branches: EnvironmentBranches, env: EnvironmentKind) {
  return env === "preview" && branches.preview.branch === branches.production.branch;
}

export function branchSummary(branches: EnvironmentBranches, env: EnvironmentKind): string {
  const { branch, compute } = branches[env];
  if (sharesProductionBranch(branches, env)) {
    return `${branch} · Same as production`;
  }
  return compute ? `${branch} · ${compute.size} · ${clusterCopy[compute.cluster].label}` : branch;
}

type BadgeVariant = ComponentProps<typeof Badge>["variant"];

export const statusCopy: Record<
  DatabaseStatus,
  { label: string; badge: BadgeVariant | null; tableBadge: BadgeVariant }
> = {
  provisioning: { label: "Provisioning", badge: "warning", tableBadge: "warning" },
  ready: { label: "Ready", badge: null, tableBadge: "success" },
};

export type PlanetscaleOrg = { name: string; hasCard: boolean };

export function orgHasCard(orgs: PlanetscaleOrg[], org: string): boolean {
  return orgs.some((o) => o.name === org && o.hasCard);
}

export type PlanetscaleDatabase = {
  name: string;
  org: string;
  engine: Engine;
  region: RegionId;
  cluster: Cluster;
  size: SizeId;
  state: DatabaseStatus;
  defaultBranch: string;
  branches: string[];
  createdAt: number;
  tableCount: number;
  schemaUpdatedAt: number | null;
  lastBackupAt: number | null;
  maintenanceAt: number | null;
  vitals: DatabaseVitals;
};

export function environmentBranch(db: PlanetscaleDatabase, branch: string): EnvironmentBranch {
  return {
    branch,
    compute: branch === db.defaultBranch ? { cluster: db.cluster, size: db.size } : null,
  };
}

const SKU_PRICES: Record<Engine, Record<Cluster, [SizeId, number][]>> = {
  postgres: {
    single: [
      ["PS-5", 5],
      ["PS-10", 10],
      ["PS-20", 17],
      ["PS-40", 28],
    ],
    ha: [
      ["PS-5", 15],
      ["PS-10", 30],
      ["PS-20", 50],
      ["PS-40", 83],
    ],
  },
  mysql: {
    single: [],
    ha: [
      ["PS-10", 39],
      ["PS-20", 59],
      ["PS-40", 99],
    ],
  },
};

export function skusFor(engine: Engine, cluster: Cluster) {
  return SKU_PRICES[engine][cluster].map(([id, monthlyPrice]) => ({
    id,
    ...sizeSpec[id],
    monthlyPrice,
  }));
}

type PlanetscaleUrl = `https://app.planetscale.com/${string}`;

export const planetscaleUrls = {
  database({ org, name }: { org: string; name: string }): PlanetscaleUrl {
    return `https://app.planetscale.com/${encodeURIComponent(org)}/${encodeURIComponent(name)}`;
  },
  settings(org: string): PlanetscaleUrl {
    return `https://app.planetscale.com/${encodeURIComponent(org)}/settings`;
  },
  billing(org: string): PlanetscaleUrl {
    return `https://app.planetscale.com/${encodeURIComponent(org)}/settings/billing`;
  },
};
