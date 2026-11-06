import type { Environment, ListedConnection } from "./target";

export type CallerEnvironment = Pick<Environment, "id" | "kind">;

export type Caller = { environment: CallerEnvironment; connections: readonly ListedConnection[] };

export type Callers<C = Caller> = { production: C; preview: C | null };

export function callerList<C>(callers: Callers<C>): C[] {
  return callers.preview ? [callers.production, callers.preview] : [callers.production];
}

export function savedIn(caller: Caller | null, targetAppId: string): ListedConnection | null {
  return caller?.connections.find((connection) => connection.targetAppId === targetAppId) ?? null;
}

export function connectedAppIds(callers: Callers): ReadonlySet<string> {
  return new Set(callers.production.connections.map((connection) => connection.targetAppId));
}
