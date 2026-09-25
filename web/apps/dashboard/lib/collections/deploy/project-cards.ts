import type { App } from "./apps";

export type ProjectApp = Pick<App, "id" | "name" | "customDomain" | "headlineDeployment">;

export function byLatestUpdate(a: CardApp, b: CardApp): number {
  return (b.updatedAt ?? -1) - (a.updatedAt ?? -1) || byIdDescending(a, b);
}

export function pickPrimaryApp<T extends CardApp>(apps: ReadonlyArray<T>): T | undefined {
  return apps.toSorted(byLatestUpdate).find((app) => app.currentDeploymentId !== null);
}

function byIdDescending(a: { id: string }, b: { id: string }): number {
  return b.id < a.id ? -1 : b.id > a.id ? 1 : 0;
}

type CardApp = Pick<App, "id" | "updatedAt" | "currentDeploymentId">;
