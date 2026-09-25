"use client";
import { collection } from "@/lib/collections";
import { warmQueries } from "@/lib/collections/warm-queries";
import { type InitialQueryBuilder, createLiveQueryCollection, eq } from "@tanstack/react-db";

export const projectAppsQueryFor = (projectId: string) => (q: InitialQueryBuilder) =>
  q
    .from({ app: collection.apps })
    .where(({ app }) => eq(app.projectId, projectId))
    .orderBy(({ app }) => app.updatedAt, { direction: "desc", nulls: "last" })
    .orderBy(({ app }) => app.id, "desc");

export function warmProjectPage(projectId: string) {
  warmQueries(`project/${projectId}`, () => [
    createLiveQueryCollection(projectAppsQueryFor(projectId)),
  ]);
}
