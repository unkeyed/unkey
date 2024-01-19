import { useRuntimeConfig } from "#imports";
import { Unkey } from "@unkey/api";
import { version } from "../../../../package.json";

import type { H3Event } from "h3";

let unkey: Unkey;

export const useUnkey = (event?: H3Event, disableTelemetry?: boolean) => {
  if (unkey) {
    return unkey;
  }

  const config = useRuntimeConfig(event);
  // TODO: allow empty tokens when registering a verification-only Unkey
  unkey = new Unkey({
    token: config.unkey.token || "invalid_token",
    wrapperSdkVersion: `@unkey/nuxt@${version}`,
    disableTelemetry,
  });

  return unkey;
};
