import type { Api } from "@unkey/db";
import { useContext } from "react";
import { useSubscribe } from "replicache-react";
import { ReplicacheContext } from "./provider";

export function useReplicache() {
  return useContext(ReplicacheContext);
}

export function useApis(): Record<string, Api> {
  const r = useReplicache();

  const res = useSubscribe(r, async (tx) => {
    const apis = {} as Record<string, Api>;
    const res = await tx.scan({ prefix: "api/" }).entries().toArray();
    for (const [k, v] of res) {
      apis[k] = v as Api;
    }
    return apis;
  });

  return res ?? {};
}
