import { composeAuthMd } from "@/lib/auth-md";
import { env } from "@/lib/env";
import { getBaseUrl } from "@/lib/utils";

export async function GET(): Promise<Response> {
  const { WORKOS_AUTHKIT_URL, UNKEY_API_URL } = env();
  if (!WORKOS_AUTHKIT_URL) {
    return new Response("Not found", { status: 404 });
  }

  const hostedUrl = new URL("/agent/auth.md", WORKOS_AUTHKIT_URL);
  let hosted: string;
  try {
    const res = await fetch(hostedUrl, {
      next: { revalidate: 300 },
      signal: AbortSignal.timeout(10_000),
    });
    if (!res.ok) {
      console.error("AuthKit auth.md fetch failed:", {
        url: hostedUrl.toString(),
        status: res.status,
      });
      return new Response("auth.md is temporarily unavailable. Retry in a minute.", {
        status: 502,
      });
    }
    hosted = await res.text();
  } catch (err) {
    console.error("AuthKit auth.md fetch failed:", err);
    return new Response("auth.md is temporarily unavailable. Retry in a minute.", { status: 502 });
  }

  return new Response(
    composeAuthMd({ hosted, apiUrl: UNKEY_API_URL, dashboardUrl: getBaseUrl() }),
    { headers: { "Content-Type": "text/markdown; charset=utf-8" } },
  );
}
