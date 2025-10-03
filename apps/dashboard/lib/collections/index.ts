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
      });
    }
    // biome-ignore lint/style/noNonNullAssertion: Its okay
    return this.projectCollections.get(projectId)!;
  }

  async preloadProject(projectId: string): Promise<void> {
    const collections = this.getProjectCollections(projectId);
    // Preload all collections in the object
    await Promise.all(Object.values(collections).map((collection) => collection.preload()));
  }

  async cleanup(projectId: string) {
    const collections = this.projectCollections.get(projectId);
    if (collections) {
      // Cleanup all collections in the object
      await Promise.all(Object.values(collections).map((collection) => collection.cleanup()));
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
