import { env } from "@/lib/env";

import { ClerkClient } from "./clerk.client";
import type { ClientAuth } from "./interface.client";
import { LocalClientAuth } from "./local.client";

export let clientAuth: ClientAuth;

let initialized = false;

function init() {
  if (initialized) {
    return;
  }

  switch (env().AUTH_PROVIDER) {
    case "clerk":
      clientAuth = new ClerkClient();
      break;
    case "local":
    case "workos":
      clientAuth = new LocalClientAuth();
      break;
  }

  initialized = true;
}

init();
