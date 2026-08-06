"use client";

/**
 * Client-side state for the permissions lab. Seeds from mock data, applies
 * edits in memory, and keeps a changeset history so concepts can share one
 * source of truth. Persisted to localStorage so edits survive navigation
 * between concept pages; "Reset lab data" clears it.
 */

import {
  type ReactNode,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
} from "react";
import {
  LEGACY_PERMISSIONS,
  type MockPrincipal,
  type MockRole,
  PRINCIPALS,
  ROLES,
} from "./mock-data";
import { type Permission, parsePermission, permissionCovers } from "./urn";

export interface GrantOp {
  op: "add" | "remove";
  principalID: string;
  /** canonical permission string */
  permission: string;
}

export interface RoleOp {
  op: "add_role" | "remove_role";
  principalID: string;
  roleID: string;
}

export type ChangeOp = GrantOp | RoleOp;

export interface Changeset {
  id: string;
  title: string;
  ops: ChangeOp[];
  status: "staged" | "applied" | "reverted";
}

export interface LabState {
  principals: MockPrincipal[];
  roles: MockRole[];
  changesets: Changeset[];
}

const STORAGE_KEY = "unkey-permissions-lab-v1";

function initialState(): LabState {
  return {
    principals: structuredClone(PRINCIPALS),
    roles: structuredClone(ROLES),
    changesets: [],
  };
}

type LabAction =
  | { type: "stage"; changeset: Changeset }
  | { type: "apply"; changesetID: string }
  | { type: "revert"; changesetID: string }
  | { type: "discard"; changesetID: string }
  | { type: "reset" }
  | { type: "hydrate"; state: LabState };

function applyOps(state: LabState, ops: ChangeOp[], invert: boolean): LabState {
  const principals = state.principals.map((p) => ({
    ...p,
    permissions: [...p.permissions],
    roles: [...p.roles],
  }));
  const byID = new Map(principals.map((p) => [p.id, p]));

  for (const op of ops) {
    const principal = byID.get(op.principalID);
    if (!principal) {
      continue;
    }
    if ("permission" in op) {
      const adding = invert ? op.op === "remove" : op.op === "add";
      if (adding) {
        if (!principal.permissions.includes(op.permission)) {
          principal.permissions.push(op.permission);
        }
      } else {
        principal.permissions = principal.permissions.filter((p) => p !== op.permission);
      }
    } else {
      const adding = invert ? op.op === "remove_role" : op.op === "add_role";
      if (adding) {
        if (!principal.roles.includes(op.roleID)) {
          principal.roles.push(op.roleID);
        }
      } else {
        principal.roles = principal.roles.filter((r) => r !== op.roleID);
      }
    }
  }
  return { ...state, principals };
}

function reducer(state: LabState, action: LabAction): LabState {
  switch (action.type) {
    case "hydrate":
      return action.state;
    case "reset":
      return initialState();
    case "stage":
      return { ...state, changesets: [action.changeset, ...state.changesets] };
    case "apply": {
      const changeset = state.changesets.find((c) => c.id === action.changesetID);
      if (!changeset || changeset.status === "applied") {
        return state;
      }
      const next = applyOps(state, changeset.ops, false);
      return {
        ...next,
        changesets: state.changesets.map((c) =>
          c.id === action.changesetID ? { ...c, status: "applied" as const } : c,
        ),
      };
    }
    case "revert": {
      const changeset = state.changesets.find((c) => c.id === action.changesetID);
      if (!changeset || changeset.status !== "applied") {
        return state;
      }
      const next = applyOps(state, changeset.ops, true);
      return {
        ...next,
        changesets: state.changesets.map((c) =>
          c.id === action.changesetID ? { ...c, status: "reverted" as const } : c,
        ),
      };
    }
    case "discard":
      return {
        ...state,
        changesets: state.changesets.filter(
          (c) => c.id !== action.changesetID || c.status !== "staged",
        ),
      };
  }
}

export interface LabStore {
  state: LabState;
  /** stage a changeset without applying it (concept: changesets) */
  stage: (title: string, ops: ChangeOp[]) => Changeset;
  /** apply a staged changeset */
  apply: (changesetID: string) => void;
  /** revert an applied changeset */
  revert: (changesetID: string) => void;
  /** discard a staged changeset */
  discard: (changesetID: string) => void;
  /** stage + apply in one step (for direct-edit concepts) */
  commit: (title: string, ops: ChangeOp[]) => void;
  reset: () => void;
  /** direct grants + grants inherited via roles, deduped */
  effectivePermissions: (principalID: string) => string[];
  /** legacy tuple permissions (exact-string match) for a principal */
  legacyPermissions: (principalID: string) => string[];
  /** does the principal cover this concrete request? */
  can: (principalID: string, request: Permission) => boolean;
}

const LabContext = createContext<LabStore | null>(null);

export function PermissionsLabProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(reducer, undefined, initialState);

  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(STORAGE_KEY);
      if (raw) {
        dispatch({ type: "hydrate", state: JSON.parse(raw) as LabState });
      }
    } catch {
      // corrupt or missing persisted state: keep the seed
    }
    // hydrate once on mount
  }, []);

  useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {
      // quota exceeded or private mode: state stays in memory
    }
  }, [state]);

  const stage = useCallback((title: string, ops: ChangeOp[]): Changeset => {
    const changeset: Changeset = {
      id: crypto.randomUUID(),
      title,
      ops,
      status: "staged",
    };
    dispatch({ type: "stage", changeset });
    return changeset;
  }, []);

  const apply = useCallback((changesetID: string) => {
    dispatch({ type: "apply", changesetID });
  }, []);

  const revert = useCallback((changesetID: string) => {
    dispatch({ type: "revert", changesetID });
  }, []);

  const discard = useCallback((changesetID: string) => {
    dispatch({ type: "discard", changesetID });
  }, []);

  const commit = useCallback(
    (title: string, ops: ChangeOp[]) => {
      const changeset = stage(title, ops);
      dispatch({ type: "apply", changesetID: changeset.id });
    },
    [stage],
  );

  const reset = useCallback(() => {
    window.localStorage.removeItem(STORAGE_KEY);
    dispatch({ type: "reset" });
  }, []);

  const store = useMemo<LabStore>(() => {
    const rolesByID = new Map(state.roles.map((r) => [r.id, r]));

    const effectivePermissions = (principalID: string): string[] => {
      const principal = state.principals.find((p) => p.id === principalID);
      if (!principal) {
        return [];
      }
      const all = new Set(principal.permissions);
      for (const roleID of principal.roles) {
        for (const p of rolesByID.get(roleID)?.permissions ?? []) {
          all.add(p);
        }
      }
      return [...all];
    };

    const legacyPermissions = (principalID: string): string[] =>
      LEGACY_PERMISSIONS[principalID] ?? [];

    const can = (principalID: string, request: Permission): boolean =>
      effectivePermissions(principalID).some((grantString) => {
        const grant = parsePermission(grantString);
        return grant.ok && permissionCovers(grant.value, request);
      });

    return {
      state,
      stage,
      apply,
      revert,
      discard,
      commit,
      reset,
      effectivePermissions,
      legacyPermissions,
      can,
    };
  }, [state, stage, apply, revert, discard, commit, reset]);

  return <LabContext.Provider value={store}>{children}</LabContext.Provider>;
}

export function usePermissionsLab(): LabStore {
  const store = useContext(LabContext);
  if (!store) {
    throw new Error("usePermissionsLab must be used inside PermissionsLabProvider");
  }
  return store;
}
