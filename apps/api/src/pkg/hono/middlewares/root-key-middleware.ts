import { UnkeyApiError } from "@/pkg/errors";
import { keyService } from "@/pkg/global";
import { MiddlewareHandler } from "hono";
import { ValidKey } from "../../keys/service";

declare module "hono" {
  interface ContextVariableMap {
    rootKey: Omit<ValidKey, "isRootKey"> & { isRootKey: boolean };
  }
}

export function rootKey(): MiddlewareHandler {
  return async (c, next) => {
    const authorization = c.req.header("authorization")!.replace("Bearer ", "");
    const rootKey = await keyService.verifyKey(c, { key: authorization });
    if (rootKey.error) {
      throw new UnkeyApiError({ code: "INTERNAL_SERVER_ERROR", message: rootKey.error.message });
    }
    if (!rootKey.value.valid) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "the root key is not valid" });
    }
    if (!rootKey.value.isRootKey) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "root key required" });
    }

    c.set("rootKey", rootKey.value);
    await next();
  };
}
