"use client";

import type { SidebarAction } from "@/lib/navigation/types";
import {
  type ReactNode,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

type ActionsMap = Record<string, SidebarAction[]>;

type Api = {
  register: (scope: string, actions: SidebarAction[]) => void;
  unregister: (scope: string) => void;
};

// State context changes on every registration. Consumers (sidebar) subscribe to it.
const SidebarActionsStateContext = createContext<SidebarAction[]>([]);
// API context is stable for the provider's lifetime. Producers (pages) subscribe to it.
const SidebarActionsApiContext = createContext<Api | null>(null);

export function SidebarActionsProvider({ children }: { children: ReactNode }) {
  const [map, setMap] = useState<ActionsMap>({});

  const register = useCallback((scope: string, actions: SidebarAction[]) => {
    setMap((prev) => ({ ...prev, [scope]: actions }));
  }, []);

  const unregister = useCallback((scope: string) => {
    setMap((prev) => {
      if (!(scope in prev)) {
        return prev;
      }
      const next = { ...prev };
      delete next[scope];
      return next;
    });
  }, []);

  const actions = useMemo(() => Object.values(map).flat(), [map]);
  const api = useMemo<Api>(() => ({ register, unregister }), [register, unregister]);

  return (
    <SidebarActionsApiContext.Provider value={api}>
      <SidebarActionsStateContext.Provider value={actions}>
        {children}
      </SidebarActionsStateContext.Provider>
    </SidebarActionsApiContext.Provider>
  );
}

export function useSidebarActionsState(): SidebarAction[] {
  return useContext(SidebarActionsStateContext);
}

// Pages call this to publish their actions to the sidebar.
// Actions are scoped by `scope` so multiple pages don't clobber each other.
export function useRegisterSidebarActions(scope: string, actions: SidebarAction[]) {
  const api = useContext(SidebarActionsApiContext);
  // Stable signature so consumers can inline the array without retriggering the effect.
  // Identity changes when label/href/disabled change; onClick identity is the caller's job.
  const signature = actions
    .map((a) => `${a.key}|${a.label}|${a.href ?? ""}|${a.disabled ?? false}`)
    .join("::");

  // Latest actions held in a ref so the effect can re-publish without depending on the
  // (always-fresh) array identity.
  const latest = useRef(actions);
  latest.current = actions;

  useEffect(() => {
    if (!api) {
      return;
    }
    api.register(scope, latest.current);
    return () => api.unregister(scope);
  }, [api, scope, signature]);
}
