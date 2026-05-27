"use client";

import { trpc } from "@/lib/trpc/client";
import { useEffect } from "react";

declare global {
  interface Window {
    uj?: {
      init: (projectId: string) => void;
      identify: (payload: UserJotIdentity | null) => void;
    };
    $ujq?: unknown[];
  }
}

type UserJotIdentity = {
  id: string;
  email: string;
  firstName: string | null;
  lastName: string | null;
  avatar: string | null;
  signature: string;
};

const SDK_SRC = "https://cdn.userjot.com/sdk/v2/uj.js";

function ensureSdkLoaded(projectId: string) {
  if (typeof window === "undefined") {
    return;
  }
  if (!window.$ujq) {
    window.$ujq = [];
    window.uj = new Proxy({} as Window["uj"] & object, {
      get:
        (_, p: string) =>
        (...args: unknown[]) =>
          window.$ujq?.push([p, ...args]),
    }) as Window["uj"];
  }
  if (!document.querySelector(`script[src="${SDK_SRC}"]`)) {
    const script = document.createElement("script");
    script.src = SDK_SRC;
    script.type = "module";
    script.async = true;
    document.head.appendChild(script);
  }
  window.uj?.init(projectId);
}

export function UserJotProvider() {
  const projectId = process.env.NEXT_PUBLIC_USERJOT_PROJECT_ID;
  const { data: identity } = trpc.user.getUserJotIdentity.useQuery(undefined, {
    enabled: Boolean(projectId),
    staleTime: 1000 * 60 * 10,
  });

  useEffect(() => {
    if (!projectId) {
      return;
    }
    ensureSdkLoaded(projectId);
  }, [projectId]);

  useEffect(() => {
    if (!projectId || typeof window === "undefined") {
      return;
    }
    if (identity) {
      window.uj?.identify(identity);
    }
  }, [projectId, identity]);

  return null;
}
