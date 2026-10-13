import { getAuth } from "@/lib/auth/get-auth";
import { mintProxyJWT } from "@/lib/auth/mint-proxy-jwt";
import { env } from "@/lib/env";
import { Unkey } from "@unkey/api";

export async function loadFlags() {
  try {
    const auth = await getAuth();
    if (!auth.userId || !auth.orgId) {
      return undefined;
    }
    let token = auth.accessToken;
    if (!token) {
      if (!auth.role) {
        throw new Error("Role required to load workspace flags");
      }
      token = await mintProxyJWT({
        orgId: auth.orgId,
        role: auth.role,
        subject: auth.userId,
        name: auth.user?.fullName ?? auth.user?.email ?? auth.userId,
      });
    }
    const client = new Unkey({
      serverURL: env().UNKEY_API_URL,
      rootKey: token,
      timeoutMs: 10_000,
      retryConfig: { strategy: "none" },
    });
    const response = await client.workspace.getFlags({ cache: "no-store", redirect: "error" });
    return response.data;
  } catch {
    console.error("Failed to load initial workspace flags; retrying on the client");
    return undefined;
  }
}
