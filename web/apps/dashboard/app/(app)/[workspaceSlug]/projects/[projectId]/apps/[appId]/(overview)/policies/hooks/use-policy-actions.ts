"use client";

import { POLICY_ERRORS, type PolicyInput } from "@/lib/collections/deploy/policies.schema";
import { type PolicyWrites, getPolicyListData, usePolicyWrites } from "@/lib/deploy-data";
import { toast } from "@unkey/ui";
import { useState } from "react";
import {
  type PolicyEdit,
  insertPolicy,
  removePolicy,
  reorderPolicies,
  replacesList,
  setEnabled,
  updatePolicy,
} from "../model/edits";
import type { Env, PolicyEnvs } from "../model/merge";

type Args = { envs: PolicyEnvs; projectId: string; appId: string };

export type PolicyActions = {
  replacingList: boolean;
  setEnabled: (key: string, env: Env, enabled: boolean) => Promise<boolean>;
  reorder: (fromKey: string, toKey: string) => Promise<boolean>;
  save: (policy: PolicyInput, envs: readonly Env[]) => Promise<boolean>;
  update: (key: string, policy: PolicyInput) => Promise<boolean>;
  delete: (key: string) => Promise<boolean>;
};

export function usePolicyActions({ envs, projectId, appId }: Args): PolicyActions {
  const writes = usePolicyWrites();
  const [replacing, setReplacing] = useState<ReadonlySet<Promise<void>>>(new Set());

  const trackReplace = (persisted: Promise<void>) => {
    setReplacing((prev) => new Set(prev).add(persisted));
    void persisted
      .catch(() => undefined)
      .finally(() =>
        setReplacing((prev) => new Set([...prev].filter((pending) => pending !== persisted))),
      );
  };

  const run = async (
    write: PolicyWrites[keyof PolicyWrites],
    edit: PolicyEdit,
  ): Promise<boolean> => {
    const production = envs.production.id;
    const preview = envs.preview.id;
    if (production === null || preview === null) {
      toast.error(POLICY_ERRORS.environmentMissing);
      return false;
    }
    const result = edit(
      { production: getPolicyListData(production), preview: getPolicyListData(preview) },
      { projectId, appId, environments: { production, preview } },
    );
    if (result.type === "reject") {
      toast.error(result.message);
      return false;
    }
    if (result.ops.length === 0) {
      return true;
    }
    const replaces = replacesList(result.ops);
    // An in-place write queued behind a list replace targets ids the replace retires.
    if (!replaces && replacing.size > 0) {
      return false;
    }
    const persisted = write(result.ops);
    if (replaces) {
      trackReplace(persisted);
    }
    return persisted.then(
      () => true,
      () => false,
    );
  };

  return {
    replacingList: replacing.size > 0,
    setEnabled: (key, env, enabled) => run(writes.update, setEnabled(key, env, enabled)),
    reorder: (fromKey, toKey) => run(writes.reorder, reorderPolicies(fromKey, toKey)),
    save: (policy, targets) => run(writes.add, insertPolicy(policy, targets)),
    update: (key, policy) => run(writes.update, updatePolicy(key, policy)),
    delete: (key) => run(writes.delete, removePolicy(key)),
  };
}
