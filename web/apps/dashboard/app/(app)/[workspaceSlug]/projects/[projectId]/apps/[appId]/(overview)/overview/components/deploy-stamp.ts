import type { GitAuthor } from "@/lib/collections/deploy/deployment-origin";
import {
  type RollbackEvent,
  type RollbackSibling,
  findReplacedDeployment,
  findRolledBackFrom,
} from "@/lib/collections/deploy/rollback";
import type { Loadable } from "@/lib/loadable/loadable";
import { match } from "@unkey/match";

export type DeployStamp = {
  verb: "Deployed" | "Rolled back";
  at: number;
  actor: RollbackEvent["actor"];
};

export type LiveState =
  | { kind: "not-live" }
  | { kind: "live"; author: GitAuthor | null }
  | { kind: "rolled-back"; event: Loadable<RollbackEvent | null> };

export type LiveFacts<T> = { stamp: DeployStamp | null; replaced: T | null };

export function liveFactsOf<T extends RollbackSibling>(
  state: LiveState,
  deployment: RollbackSibling,
  deployments: readonly T[],
): LiveFacts<T> {
  return match(state)
    .returnType<LiveFacts<T>>()
    .with({ kind: "not-live" }, () => ({ stamp: null, replaced: null }))
    .with({ kind: "live" }, ({ author }) => ({
      stamp: {
        verb: "Deployed",
        at: deployment.createdAt,
        actor: author?.handle ? { name: author.handle, avatarUrl: author.avatarUrl } : null,
      },
      replaced: null,
    }))
    .with({ kind: "rolled-back" }, ({ event }) => {
      const inferred = () => ({
        stamp: null,
        replaced: findRolledBackFrom(deployments, deployment) ?? null,
      });
      return match(event)
        .returnType<LiveFacts<T>>()
        .with({ status: "loading" }, () => ({ stamp: null, replaced: null }))
        .with({ status: "failed" }, inferred)
        .with({ status: "ready" }, ({ data }) =>
          data
            ? {
                stamp: { verb: "Rolled back", at: data.at, actor: data.actor },
                replaced:
                  findReplacedDeployment(deployments, deployment, data.fromDeploymentId) ?? null,
              }
            : inferred(),
        )
        .exhaustive();
    })
    .exhaustive();
}
