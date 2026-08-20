"use client";

import { getErrorMessage, getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import type { Portal } from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import { getPortalByMapping, keyspaceMapping } from "./client";

type CreatePortalInput = Parameters<Unkey["portal"]["createPortal"]>[0];
type CreatePortalResult = Awaited<ReturnType<Unkey["portal"]["createPortal"]>>["data"];
type UpdatePortalInput = Parameters<Unkey["portal"]["updatePortal"]>[0];
type UpdatePortalResult = Awaited<ReturnType<Unkey["portal"]["updatePortal"]>>["data"];
type DeletePortalInput = Parameters<Unkey["portal"]["deletePortal"]>[0];
type DeletePortalResult = Awaited<ReturnType<Unkey["portal"]["deletePortal"]>>["data"];

/**
 * The five states a keyspace's portal can be in. `notConfigured` and `disabled`
 * are distinct: the API keeps the row when a portal is disabled, and the surface
 * has to offer re-enable rather than create. `error` exists so a transient
 * failure is never mistaken for "no portal yet".
 */
export type PortalState =
  | { status: "loading" }
  | { status: "notConfigured" }
  | { status: "disabled"; portal: Portal }
  | { status: "enabled"; portal: Portal }
  | { status: "error"; message: string };

/** A 404 means "no portal for this keyspace", which is a state, not a failure. */
export type PortalQueryResult = { found: true; portal: Portal } | { found: false };

export function portalQueryKey(keyAuthId: string): readonly [string, string] {
  return ["portal", keyAuthId];
}

export function usePortal(keyAuthId: string | undefined): PortalState {
  const query = useQuery<PortalQueryResult>({
    queryKey: portalQueryKey(keyAuthId ?? ""),
    enabled: Boolean(keyAuthId),
    queryFn: async () => {
      if (!keyAuthId) {
        // `enabled` gates the query on a keyspace id, so this is unreachable.
        // Reporting `found: false` here would spell "no portal" for what is
        // really "no keyspace" — two states the surface treats differently.
        throw new Error("portal query ran without a keyspace id");
      }
      try {
        const portal = await getPortalByMapping(keyspaceMapping(keyAuthId));
        return { found: true, portal };
      } catch (error) {
        if (error instanceof NotFoundErrorResponse) {
          return { found: false };
        }
        throw error;
      }
    },
  });

  // Only surface the error when there is nothing to render. The provider
  // refetches on window focus with `retry: 1`, so a single failed background
  // refetch over a perfectly good cached row must not unmount the configuration
  // view and throw away in-progress branding or slug edits.
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

/**
 * Lets a caller claim an error — a slug conflict belongs on the form field, not
 * in a toast. Return true to suppress the default toast.
 */
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

/**
 * Creates the portal for this keyspace. The mapping is supplied here rather than
 * by the caller, so this surface can only ever create a keyspace portal.
 */
export function useCreatePortal(keyAuthId: string, options?: PortalMutationOptions) {
  const invalidate = useInvalidatePortal(keyAuthId);

  return useMutation<CreatePortalResult, unknown, Omit<CreatePortalInput, "mapping">>({
    mutationFn: async (input) => {
      const response = await getUnkeyClient().portal.createPortal({
        ...input,
        mapping: keyspaceMapping(keyAuthId),
      });
      return response.data;
    },
    onSuccess: invalidate,
    onError: toastUnless(options, "Failed to Create Portal"),
  });
}

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
