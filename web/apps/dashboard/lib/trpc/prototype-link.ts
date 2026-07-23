import { TRPCClientError, type TRPCLink } from "@trpc/client";
import { observable } from "@trpc/server/observable";
import type { Router } from "@/lib/trpc/routers";

// Prototype-only data layer for the vibe branch: intercepts a fixed set of
// tRPC procedures on the client and serves deterministic data from the
// localStorage-backed prototype store, so the real dashboard UIs (keyspace
// list/detail, keys pages, ratelimits) render fully without any backend or
// ClickHouse seed. Everything else passes through to the real API.

// Sentinel a replace handler can return to fall through to the real backend
// (used for procedures that are only faked for prototype-owned ids).
export const PASS: unique symbol = Symbol("prototype-pass");

export type ReplaceHandler = (input: unknown) => unknown;
// Merge handlers receive the real backend response (or null when it failed)
// and return the combined result.
export type MergeHandler = (input: unknown, real: unknown | null) => unknown;

export type PrototypeHandlers = {
  replace: Record<string, ReplaceHandler>;
  merge: Record<string, MergeHandler>;
};

function toClientError(err: unknown): TRPCClientError<Router> {
  return TRPCClientError.from(err instanceof Error ? err : new Error(String(err)));
}

export function createPrototypeLink(handlers: PrototypeHandlers): TRPCLink<Router> {
  return () =>
    ({ op, next }) => {
      if (op.type === "subscription") {
        return next(op);
      }

      const replace = handlers.replace[op.path];
      if (replace) {
        let data: unknown;
        try {
          data = replace(op.input);
        } catch (err) {
          return observable((observer) => {
            observer.error(toClientError(err));
          });
        }
        if (data !== PASS) {
          return observable((observer) => {
            observer.next({ result: { type: "data", data } });
            observer.complete();
          });
        }
      }

      const merge = handlers.merge[op.path];
      if (merge) {
        return observable((observer) => {
          const sub = next(op).subscribe({
            next(envelope) {
              const data =
                envelope.result && "data" in envelope.result ? envelope.result.data : null;
              observer.next({
                ...envelope,
                result: { type: "data", data: merge(op.input, data) },
              });
            },
            error() {
              // Backend unavailable (e.g. fresh preview workspace) — still
              // serve the prototype rows so the page renders.
              observer.next({ result: { type: "data", data: merge(op.input, null) } });
              observer.complete();
            },
            complete() {
              observer.complete();
            },
          });
          return () => {
            sub.unsubscribe();
          };
        });
      }

      return next(op);
    };
}
