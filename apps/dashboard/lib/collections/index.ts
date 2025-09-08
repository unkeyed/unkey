"use client";

import { ratelimitNamespaces } from "./ratelimit_namespaces";
import { ratelimitOverrides } from "./ratelimit_overrides";


export const collection = {
  ratelimitNamespaces,
  ratelimitOverrides,
};

// resets all collections data and preloads new
export function reset() {
  for (const c of [ratelimitNamespaces, ratelimitOverrides]) {
    c.cleanup();
    c.preload();
  }
}
