import { relations, sql } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  json,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { deployments } from "./deployments";
import { projects } from "./projects";
import { regions } from "./regions";

//id, deplyoment_id, health, kube_dns_addr, mem, cpu, region

// ContainerStatus mirrors the kubelet-detailed shape we get from krane's
// pod-watch reports. Stored as JSON so adding a new state kind (e.g.
// imagePullState, init container progress) is a code-only change with no
// migration. The flat lifecycle enum on `status` is a separate concern —
// it answers "did this pod come up?", while ContainerStatus answers
// "what's the container actually doing right now?".
//
// Shape mirrors corev1.ContainerStatus:
//   - restartCount: monotonic counter of kubelet-observed restarts
//   - lastTerminationState: most recent exit (absent when never exited)
//   - waiting: current kubelet waiting reason (absent when running normally)
//
// Reads project keys directly: `status->>'$.lastTerminationState.exitCode'`.
// Writes use JSON_SET to overwrite specific keys atomically. The dashboard
// can iterate keys to render an "array of statuses" without us actually
// storing one.
export type ContainerStatus = {
  restartCount: number;
  lastTerminationState?: {
    exitCode: number;
    signal: number;
    reason: string;
    finishedAt: number; // unix milliseconds
  };
  waiting?: {
    reason: string; // CrashLoopBackOff, ImagePullBackOff, ContainerCreating, …
  };
};

export const instances = mysqlTable(
  "instances",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    deploymentId: varchar("deployment_id", { length: 255 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    projectId: varchar("project_id", { length: 255 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),

    regionId: varchar("region_id", { length: 64 }).notNull(),

    // used to apply updates from the kubernetes watch events
    k8sName: varchar("k8s_name", { length: 255 }).notNull(),
    // The kubernetes pod dns address. Not uniquely constrained per region because
    // Kubernetes recycles pod IPs across deployments.
    address: varchar("address", { length: 255 }).notNull(),
    cpuMillicores: int("cpu_millicores").notNull(),
    memoryMib: int("memory_mib").notNull(),
    storageMib: int("storage_mib", { unsigned: true }).notNull().default(0),
    status: mysqlEnum("status", ["inactive", "pending", "running", "failed"]).notNull(),
    // Kubelet-detailed container status; see [ContainerStatus] above.
    // Default ships restartCount=0 so the not-null guarantee holds without
    // every read having to defend against a missing key.
    containerStatus: json("container_status")
      .$type<ContainerStatus>()
      .notNull()
      .default(sql`(JSON_OBJECT('restartCount', 0))`),
  },
  (table) => [
    uniqueIndex("unique_k8s_name_per_region").on(table.k8sName, table.regionId),
    index("idx_deployment_id").on(table.deploymentId),
    index("idx_region").on(table.regionId),
  ],
);

export const instancesRelations = relations(instances, ({ one }) => ({
  deployment: one(deployments, {
    fields: [instances.deploymentId],
    references: [deployments.id],
  }),
  project: one(projects, {
    fields: [instances.projectId],
    references: [projects.id],
  }),
  region: one(regions, {
    fields: [instances.regionId],
    references: [regions.id],
  }),
}));
