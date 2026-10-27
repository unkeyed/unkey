"use client";

import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { toast } from "@unkey/ui";
import {
  listNotificationDestinations,
  listNotificationMembers,
  listNotificationRules,
} from "./client";

export const notificationKeys = {
  workspace: (workspaceId: string, userId: string) =>
    ["notifications", workspaceId, userId] as const,
};

export function notificationError(error: unknown, fallback: string) {
  const { message, description } = getErrorToast(error, fallback);
  toast.error(message, { description });
}

export function useNotificationSession() {
  const { workspace, user, isLoading } = useWorkspace();
  return {
    user,
    workspace,
    isLoading,
    ready: Boolean(workspace && user),
    canManage: Boolean(workspace && user?.role === "admin"),
    key: notificationKeys.workspace(workspace?.id ?? "", user?.id ?? ""),
  };
}

export function useNotificationCatalog() {
  const session = useNotificationSession();
  return useQuery({
    queryKey: [...session.key, "catalog"],
    enabled: session.ready,
    queryFn: async ({ signal }) =>
      (await getUnkeyClient().notifications.getCatalog({ signal })).data,
  });
}

export function useNotificationRules() {
  const session = useNotificationSession();
  return useQuery({
    queryKey: [...session.key, "rules"],
    enabled: session.canManage,
    queryFn: ({ signal }) => listNotificationRules(signal),
  });
}

export function useNotificationDestinations() {
  const session = useNotificationSession();
  return useQuery({
    queryKey: [...session.key, "destinations"],
    enabled: session.canManage,
    queryFn: ({ signal }) => listNotificationDestinations(signal),
  });
}

export function useNotificationMembers() {
  const session = useNotificationSession();
  return useQuery({
    queryKey: [...session.key, "members"],
    enabled: session.canManage,
    queryFn: ({ signal }) => listNotificationMembers(signal),
  });
}

export function useNotificationPreferences() {
  const session = useNotificationSession();
  const queryClient = useQueryClient();
  const key = [...session.key, "preferences"];
  const query = useQuery({
    queryKey: key,
    enabled: session.ready && !session.user?.impersonator,
    queryFn: async ({ signal }) =>
      (await getUnkeyClient().notifications.getPreferences({ signal })).data,
  });
  const mutation = useMutation({
    mutationFn: async (input: Parameters<Unkey["notifications"]["updatePreferences"]>[0]) =>
      (await getUnkeyClient().notifications.updatePreferences(input)).data,
    onSuccess: async (data) => {
      await queryClient.cancelQueries({ queryKey: key });
      queryClient.setQueryData(key, data);
      await queryClient.invalidateQueries({ queryKey: [...session.key, "recipients"] });
    },
    onError: (error) => notificationError(error, "Couldn't save email preferences"),
  });
  return { query, mutation };
}

export function useNotificationRecipients(
  input: Parameters<Unkey["notifications"]["previewRecipients"]>[0] | null,
) {
  const session = useNotificationSession();
  return useQuery({
    queryKey: [...session.key, "recipients", input],
    enabled: session.canManage && input !== null,
    queryFn: async ({ signal }) => {
      if (!input) {
        throw new Error("Recipient preview requires a concrete resource");
      }
      return (await getUnkeyClient().notifications.previewRecipients(input, { signal })).data;
    },
  });
}
