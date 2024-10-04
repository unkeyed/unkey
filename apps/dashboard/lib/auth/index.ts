import { env } from "@/lib/env";
 
import type { Auth } from "./interface";
import { LocalAuth } from "./local";
import { WorkosAuth } from "./workos";
 
export let auth: Auth<any>;
 
let initialized = false;
 
function init(): void {
  if (initialized) {
    return;
  }
 
  const authProvider = env().AUTH_PROVIDER;

  switch (authProvider) {
    case "workos":
     auth = new WorkosAuth();
      break;
    case "local":
      auth = new LocalAuth();
      break;
    default:
        throw new Error(`Unsupported AUTH_PROVIDER: ${authProvider}`)
  }
 
  initialized = true;
}
 
init();