"use client";

import { RadarSignalsProvider, useRadarToken } from "@workos/radar-signals/react";
import { type ReactNode, createContext, useCallback, useContext, useMemo } from "react";

type RadarSignals = {
  /**
   * Returns the collected browser-signal token, or undefined when Radar is
   * not active (local auth / non-WorkOS) or collection failed. Always safe to
   * call and always fails open, so client-side signal collection can never
   * block a sign-in attempt.
   */
  getToken: () => Promise<string | undefined>;
  /** True once a real token is available; always true in the no-op case. */
  tokenReady: boolean;
};

// Default value used when Radar is not mounted (local / non-WorkOS). Consumers
// can therefore call useRadarSignals() unconditionally without a provider.
const RadarSignalsContext = createContext<RadarSignals>({
  getToken: async () => undefined,
  tokenReady: true,
});

// Republishes WorkOS's useRadarToken through our own context so the rest of the
// auth flow has a single, always-available hook regardless of whether Radar is
// mounted. Must live inside RadarSignalsProvider.
function RadarSignalsBridge({ children }: { children: ReactNode }) {
  const { getToken, tokenReady } = useRadarToken();

  const safeGetToken = useCallback(async () => {
    try {
      return await getToken();
    } catch {
      // Fail open: a collection error must never stop the user signing in.
      return undefined;
    }
  }, [getToken]);

  const value = useMemo<RadarSignals>(
    () => ({ getToken: safeGetToken, tokenReady }),
    [safeGetToken, tokenReady],
  );

  return <RadarSignalsContext.Provider value={value}>{children}</RadarSignalsContext.Provider>;
}

/**
 * Mounts WorkOS Radar browser-signal collection for the auth flows, but only
 * when a publishable WorkOS client id is configured. Local development and
 * self-hosters on AUTH_PROVIDER="local" never set NEXT_PUBLIC_WORKOS_CLIENT_ID,
 * so the Radar CDN script is never loaded and useRadarSignals() degrades to a
 * no-op that resolves to undefined.
 */
export function RadarProvider({ children }: { children: ReactNode }) {
  const clientId = process.env.NEXT_PUBLIC_WORKOS_CLIENT_ID;
  if (!clientId) {
    return <>{children}</>;
  }

  return (
    <RadarSignalsProvider clientId={clientId}>
      <RadarSignalsBridge>{children}</RadarSignalsBridge>
    </RadarSignalsProvider>
  );
}

export function useRadarSignals(): RadarSignals {
  return useContext(RadarSignalsContext);
}
