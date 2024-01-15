import { useRuntimeConfig } from "#imports";
import s from "@nuxt/schema";
import { Unkey } from "@unkey/api";
import type { H3Event } from "h3";

// @nuxt-schema is required for the build to pass, but knip doesn't understand why.
// I need to use it for now to show that it's required - I can't ignore the package.json.
// TODO: fix the issues with our Nuxt types
const _schema = s;

let unkey: Unkey;

export const useUnkey = (event?: H3Event) => {
  if (unkey) {
    return unkey;
  }

  const config = useRuntimeConfig(event);
  // TODO: allow empty tokens when registering a verification-only Unkey
  unkey = new Unkey({ token: config.unkey.token || "invalid_token" });

  return unkey;
};
