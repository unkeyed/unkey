import { getAuth } from "@/lib/auth/get-auth";
import { vercelAdapter } from "@flags-sdk/vercel";
import type { Adapter } from "flags";
import { dedupe } from "flags/next";

// Entities contains only stable targeting identifiers. Avoid adding session
// details here unless the flag provider needs them for evaluation.
export type Entities = {
  user?: { id: string };
  org?: { id: string };
};

// identify uses the non-redirecting auth helper because flag evaluation can run
// during render paths where redirecting on anonymous users would break the page.
export const identify = dedupe(async (): Promise<Entities> => {
  const { userId, orgId } = await getAuth();
  return {
    user: userId ? { id: userId } : undefined,
    org: orgId ? { id: orgId } : undefined,
  };
});

// adapter creates the Vercel Flags adapter and falls back to a local adapter
// when the Vercel FLAGS setup is absent. Local flag evaluation must not prevent
// the dashboard from booting.
export function adapter<T>(): Adapter<T, Entities> {
  // Local escape hatch: set FLAGS_LOCAL_ALL_ON=true to resolve every flag to
  // "on" without configuring Vercel Flags locally. All flags today are
  // booleans, so "on" is `true`. Never set this in production.
  if (process.env.FLAGS_LOCAL_ALL_ON === "true") {
    return {
      config: { reportValue: false },
      decide: () => true as T,
    };
  }

  try {
    return vercelAdapter();
  } catch {
    console.warn("[flags] failed to create vercel adapter, falling back to noop");
    return {
      config: { reportValue: false },
      decide: ({ defaultValue }) => {
        if (defaultValue !== undefined) {
          return defaultValue;
        }

        throw new Error("[flags] noop adapter requires a defaultValue");
      },
    };
  }
}
