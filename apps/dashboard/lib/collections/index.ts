"use client";
import { createDeploymentsCollection } from "./deploy/deployments";
import { createDomainsCollection } from "./deploy/domains";
import { createEnvironmentsCollection } from "./deploy/environments";
import { createProjectsCollection } from "./deploy/projects";
import { createRatelimitNamespacesCollection } from "./ratelimit/namespaces";
import { createRatelimitOverridesCollection } from "./ratelimit/overrides";

// Export types
export type { Deployment } from "./deploy/deployments";
export type { Domain } from "./deploy/domains";
export type { Project } from "./deploy/projects";
export type { RatelimitNamespace } from "./ratelimit/namespaces";
export type { RatelimitOverride } from "./ratelimit/overrides";
export type { Environment } from "./deploy/environments";

// Collection factory definitions - only project-scoped collections
const PROJECT_COLLECTION_FACTORIES = {
  environments: createEnvironmentsCollection,
  domains: createDomainsCollection,
  deployments: createDeploymentsCollection,
} as const;

const GLOBAL_COLLECTION_FACTORIES = {
  projects: createProjectsCollection,
  ratelimitNamespaces: createRatelimitNamespacesCollection,
  ratelimitOverrides: createRatelimitOverridesCollection,
} as const;

// ProjectCollections only contains project-scoped collections
type ProjectCollections = {
  [K in keyof typeof PROJECT_COLLECTION_FACTORIES]: ReturnType<
    (typeof PROJECT_COLLECTION_FACTORIES)[K]
  >;
};

async function cleanupCollections(collections: Record<string, { cleanup(): Promise<void> }>) {
  await Promise.all(Object.values(collections).map((c) => c.cleanup()));
}

class CollectionManager {
  private projectCollections = new Map<string, ProjectCollections>();

  getProjectCollections(projectId: string): ProjectCollections {
    if (!projectId) {
      throw new Error("projectId is required");
    }

    if (!this.projectCollections.has(projectId)) {
      // Create collections using factories - only project-scoped ones
      const newCollections = Object.fromEntries(
        Object.entries(PROJECT_COLLECTION_FACTORIES).map(([key, factory]) => [
          key,
          factory(projectId),
        ]),
      ) as ProjectCollections;

      this.projectCollections.set(projectId, newCollections);
    }
    // biome-ignore lint/style/noNonNullAssertion: Its okay
    return this.projectCollections.get(projectId)!;
  }

  async cleanup(projectId: string) {
    const collections = this.projectCollections.get(projectId);
    if (collections) {
      // All collections in ProjectCollections are cleanupable
      await cleanupCollections(collections);
      this.projectCollections.delete(projectId);
    }
  }

  async cleanupAll() {
    // Clean up all project collections
    const projectPromises = Array.from(this.projectCollections.entries()).map(
      async ([_, collections]) => {
        await cleanupCollections(collections);
      },
    );
    // Clean up global collections, this has to run sequentially
    for (const c of Object.values(collection)) {
      await c.cleanup();
    }

    await Promise.all([...projectPromises]);
    this.projectCollections.clear();
  }
}

export const collectionManager = new CollectionManager();

// Global collections, create using factories
export const collection = Object.fromEntries(
  Object.entries(GLOBAL_COLLECTION_FACTORIES).map(([key, factory]) => [key, factory()]),
) as {
  [K in keyof typeof GLOBAL_COLLECTION_FACTORIES]: ReturnType<
    (typeof GLOBAL_COLLECTION_FACTORIES)[K]
  >;
};

export async function reset() {
  // This is GC cleanup only useful for better memory management
  await collectionManager.cleanupAll();
  // Without these components still subscribed to old collections, so create new instances for each reset. Mostly used when switching workspaces
  Object.assign(
    collection,
    Object.fromEntries(
      Object.entries(GLOBAL_COLLECTION_FACTORIES).map(([key, factory]) => [key, factory()]),
    ),
  );
  // Preload all collections, please keep this sequential. Otherwise UI acts weird. react-query already takes care of batching.
  for (const c of Object.values(collection)) {
    await c.preload();
  }
}
