"use client";

import { useCallback, useReducer, useRef, useState } from "react";
import type { Env, MergedPolicy, PolicyRowKey } from "../components/list/merge";
import {
  NO_PENDING_SWITCHES,
  type PolicySwitches,
  policySwitches,
  reducePendingSwitches,
} from "./policy-switches";

export type PolicySwitchControls = {
  switchesOf: (policy: MergedPolicy) => PolicySwitches;
  toggle: (key: PolicyRowKey, env: Env) => void;
};

export function usePolicySwitches(
  merged: MergedPolicy[],
  setEnabled: (key: PolicyRowKey, env: Env, on: boolean) => Promise<boolean>,
): PolicySwitchControls {
  const [pending, dispatch] = useReducer(reducePendingSwitches, NO_PENDING_SWITCHES);
  const [synced, setSynced] = useState(merged);
  if (merged !== synced) {
    setSynced(merged);
    dispatch({ type: "sync" });
  }
  const clicks = useRef(0);

  const switchesOf = useCallback(
    (policy: MergedPolicy) => policySwitches(policy, pending),
    [pending],
  );

  const toggle = useCallback(
    (key: PolicyRowKey, env: Env) => {
      const policy = merged.find((m) => m.key === key);
      if (!policy) {
        return;
      }
      const on = !policySwitches(policy, pending)[env];
      clicks.current += 1;
      const click = clicks.current;
      dispatch({ type: "click", key, env, on, click });
      void setEnabled(key, env, on).then((ok) => dispatch({ type: "settle", key, env, click, ok }));
    },
    [merged, pending, setEnabled],
  );

  return { switchesOf, toggle };
}
