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

// Local escape hatch: set FLAGS_LOCAL_OVERRIDES to a comma-separated list of
// flag keys to force them "on" without configuring Vercel Flags locally (e.g.
// FLAGS_LOCAL_OVERRIDES="deploy-billing,hello-world"). All flags today are
// booleans, so "on" is `true`. An override wins over the underlying adapter, so
// per-flag prod defaults stay honest. Never set this in production.
const localOverrides = new Set(
  (process.env.FLAGS_LOCAL_OVERRIDES ?? "")
    .split(",")
    .map((key) => key.trim())
    .filter((key) => key.length > 0),
);

// adapter creates the Vercel Flags adapter and falls back to a local adapter
// when the Vercel FLAGS setup is absent. Local flag evaluation must not prevent
// the dashboard from booting.
export function adapter<T>(): Adapter<T, Entities> {
  let base: Adapter<T, Entities>;
  try {
    base = vercelAdapter();
  } catch {
    console.warn("[flags] failed to create vercel adapter, falling back to noop");
    base = {
      config: { reportValue: false },
      decide: ({ defaultValue }) => {
        if (defaultValue !== undefined) {
          return defaultValue;
        }

        throw new Error("[flags] noop adapter requires a defaultValue");
      },
    };
  }

  if (localOverrides.size === 0) {
    return base;
  }

  return {
    ...base,
    decide: (params) => (localOverrides.has(params.key) ? (true as T) : base.decide(params)),
  };
}
