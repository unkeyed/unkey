import { env } from "@/lib/env";

import { ClerkServerAuth } from "./clerk.server";
import type { ServerAuth } from "./interface.server";
import { LocalServerAuth } from "./local.server";
import { WorkosServerAuth } from "./workos.server";

export let serverAuth: ServerAuth;

let initialized = false;

function init() {
  if (initialized) {
    return;
  }

  switch (env().AUTH_PROVIDER) {
    case "workos":
      serverAuth = new WorkosServerAuth();
      break;
    case "clerk":
      serverAuth = new ClerkServerAuth();
      break;
    case "local":
      serverAuth = new LocalServerAuth();
      break;
  }

  initialized = true;
}

init();
