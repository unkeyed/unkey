"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import type { Portal } from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import { getErrorMessage, getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { getPortalByKeyspace } from "./client";

type CreatePortalInput = Parameters<Unkey["portal"]["createPortal"]>[0];
type CreatePortalResult = Awaited<ReturnType<Unkey["portal"]["createPortal"]>>["data"];
type UpdatePortalInput = Parameters<Unkey["portal"]["updatePortal"]>[0];
type UpdatePortalResult = Awaited<ReturnType<Unkey["portal"]["updatePortal"]>>["data"];
type DeletePortalInput = Parameters<Unkey["portal"]["deletePortal"]>[0];
type DeletePortalResult = Awaited<ReturnType<Unkey["portal"]["deletePortal"]>>["data"];

// `notConfigured` and `disabled` are distinct because the API keeps the row when
// a portal is disabled, so the surface offers re-enable rather than create.
export type PortalState =
  | { status: "loading" }
  | { status: "notConfigured" }
  | { status: "disabled"; portal: Portal }
  | { status: "enabled"; portal: Portal }
  | { status: "error"; message: string };

// A 404 means "no portal for this keyspace", which is a state, not a failure.
export type PortalQueryResult = { found: true; portal: Portal } | { found: false };

// Shared by the read and every mutation's invalidation, so they cannot drift.
export function portalQueryKey(keyAuthId: string): readonly [string, string] {
  return ["portal", keyAuthId];
}

// Collapses the read into the states the surface renders. Undefined keyspace id
// means the query never runs.
export function usePortal(keyAuthId: string | undefined): PortalState {
  const query = useQuery<PortalQueryResult>({
    queryKey: portalQueryKey(keyAuthId ?? ""),
    enabled: Boolean(keyAuthId),
    queryFn: async () => {
      if (!keyAuthId) {
        // Unreachable via `enabled`. Throwing rather than `found: false`, which
        // would spell "no portal" for what is really "no keyspace".
        throw new Error("portal query ran without a keyspace id");
      }
      try {
        const portal = await getPortalByKeyspace(keyAuthId);
        return { found: true, portal };
      } catch (error) {
        if (error instanceof NotFoundErrorResponse) {
          return { found: false };
        }
        throw error;
      }
    },
  });

  // Only surface the error when there is no cached row: the provider refetches
  // on window focus, and unmounting the config view on a failed background
  // refetch would throw away in-progress edits.
  if (!query.data) {
    if (query.error) {
      return { status: "error", message: getErrorMessage(query.error) };
    }
    return { status: "loading" };
  }

  if (!query.data.found) {
    return { status: "notConfigured" };
  }

  const { portal } = query.data;
  return { status: portal.enabled ? "enabled" : "disabled", portal };
}

// `onError` returns true to claim an error, suppressing the default toast so
// the caller can render it in place, such as on a form field.
export type PortalMutationOptions = {
  onError?: (error: unknown) => boolean;
};

function useInvalidatePortal(keyAuthId: string) {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: portalQueryKey(keyAuthId) });
}

function toastUnless(options: PortalMutationOptions | undefined, fallback: string) {
  return (error: unknown) => {
    if (options?.onError?.(error)) {
      return;
    }
    const { message, description } = getErrorToast(error, fallback);
    toast.error(message, { description });
  };
}

// Creates the portal for this keyspace. The keyspace id is supplied here rather
// than by the caller, so this surface can only ever create a keyspace portal.
export function useCreatePortal(keyAuthId: string, options?: PortalMutationOptions) {
  const invalidate = useInvalidatePortal(keyAuthId);

  // `keyspaceId` is supplied here, and the request union leaves `appId` optional
  // on this arm, so excluding both keeps a caller from sending a shape only the
  // server would reject.
  return useMutation<CreatePortalResult, unknown, Omit<CreatePortalInput, "keyspaceId" | "appId">>({
    mutationFn: async (input) => {
      const response = await getUnkeyClient().portal.createPortal({
        ...input,
        keyspaceId: keyAuthId,
      });
      return response.data;
    },
    onSuccess: invalidate,
    onError: toastUnless(options, "Failed to Create Portal"),
  });
}

// Patches the portal. Callers pass only the fields the operator edited.
export function useUpdatePortal(keyAuthId: string, options?: PortalMutationOptions) {
  const invalidate = useInvalidatePortal(keyAuthId);

  return useMutation<UpdatePortalResult, unknown, UpdatePortalInput>({
    mutationFn: async (input) => {
      const response = await getUnkeyClient().portal.updatePortal(input);
      return response.data;
    },
    onSuccess: invalidate,
    onError: toastUnless(options, "Failed to Update Portal"),
  });
}

// Deletes the portal and revokes its live sessions.
export function useDeletePortal(keyAuthId: string, options?: PortalMutationOptions) {
  const invalidate = useInvalidatePortal(keyAuthId);

  return useMutation<DeletePortalResult, unknown, DeletePortalInput>({
    mutationFn: async (input) => {
      const response = await getUnkeyClient().portal.deletePortal(input);
      return response.data;
    },
    onSuccess: invalidate,
    onError: toastUnless(options, "Failed to Delete Portal"),
  });
}
