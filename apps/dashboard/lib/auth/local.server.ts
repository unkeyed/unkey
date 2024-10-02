import type { ServerAuth } from "./interface.server";

const TENANT_ID = "user_local";

export class LocalServerAuth implements ServerAuth {
  async getTenantId() {
    return Promise.resolve(TENANT_ID);
  }

  async getUser() {
    return Promise.resolve({ id: "user_123" });
  }
}
