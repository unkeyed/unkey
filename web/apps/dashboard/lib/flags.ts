// Use the non-redirecting variant; @/lib/auth.ts redirects on unauth, which would break flag eval.
import { getAuth } from "@/lib/auth/get-auth";
import { vercelAdapter } from "@flags-sdk/vercel";
import { dedupe, flag } from "flags/next";

interface Entities {
  user?: { id: string };
  org?: { id: string };
}

const identify = dedupe(async (): Promise<Entities> => {
  const { userId, orgId } = await getAuth();
  return {
    user: userId ? { id: userId } : undefined,
    org: orgId ? { id: orgId } : undefined,
  };
});

export const helloWorld = flag<boolean, Entities>({
  key: "hello-world",
  description: "Smoke test for the flags pipeline",
  defaultValue: false,
  options: [
    { value: false, label: "Off" },
    { value: true, label: "On" },
  ],
  identify,
  adapter: vercelAdapter(),
});
