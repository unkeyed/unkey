"use client";
import { createDeploymentsCollection } from "./deploy/deployments";
import { createDomainsCollection } from "./deploy/domains";
import { createEnvironmentsCollection } from "./deploy/environments";
import { projects } from "./deploy/projects";
import { ratelimitNamespaces } from "./ratelimit/namespaces";
import { ratelimitOverrides } from "./ratelimit/overrides";

// Export types
export type { Deployment } from "./deploy/deployments";
export type { Domain } from "./deploy/domains";
export type { Project } from "./deploy/projects";
export type { RatelimitNamespace } from "./ratelimit/namespaces";
export type { RatelimitOverride } from "./ratelimit/overrides";
export type { Environment } from "./deploy/environments";

type ProjectCollections = {
  environments: ReturnType<typeof createEnvironmentsCollection>;
  domains: ReturnType<typeof createDomainsCollection>;
  deployments: ReturnType<typeof createDeploymentsCollection>;
  projects: typeof projects;
};

class CollectionManager {
  private projectCollections = new Map<string, ProjectCollections>();

  getProjectCollections(projectId: string): ProjectCollections {
    if (!projectId) {
      throw new Error("projectId is required");
    }
    if (!this.projectCollections.has(projectId)) {
      this.projectCollections.set(projectId, {
        environments: createEnvironmentsCollection(projectId),
        domains: createDomainsCollection(projectId),
        deployments: createDeploymentsCollection(projectId),
        projects,
      });
    }
    // biome-ignore lint/style/noNonNullAssertion: Its okay
    return this.projectCollections.get(projectId)!;
  }

  async cleanup(projectId: string) {
    const collections = this.projectCollections.get(projectId);
    if (collections) {
      await Promise.all([
        collections.environments.cleanup(),
        collections.domains.cleanup(),
        collections.deployments.cleanup(),
        // Note: projects is shared, don't clean it up per project
      ]);
      this.projectCollections.delete(projectId);
    }
  }

  async cleanupAll() {
    // Clean up all project collections
    const projectCleanupPromises = Array.from(this.projectCollections.keys()).map((projectId) =>
      this.cleanup(projectId),
    );

    // Clean up global collections
    const globalCleanupPromises = Object.values(collection).map((c) => c.cleanup());

    await Promise.all([...projectCleanupPromises, ...globalCleanupPromises]);
  }
}

export const collectionManager = new CollectionManager();

// Global collections
export const collection = {
  projects,
  ratelimitNamespaces,
  ratelimitOverrides,
} as const;

export async function reset() {
  await collectionManager.cleanupAll();
  // Preload global collections after cleanup
  await Promise.all(
    Object.values(collection).map(async (c) => {
      await c.preload();
    }),
  );
}
