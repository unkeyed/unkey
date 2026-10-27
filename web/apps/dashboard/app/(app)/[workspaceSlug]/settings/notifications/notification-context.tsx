"use client";

import {
  useNotificationCatalog,
  useNotificationDestinations,
  useNotificationMembers,
  useNotificationRules,
  useNotificationSession,
} from "@/lib/notifications/use-notifications";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useQueryClient } from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { Button } from "@unkey/ui";
import { type ReactNode, createContext, useContext } from "react";
import { type Rule, ruleFields, toRuleModel } from "./notification-model";
import { supportedNotificationTypes } from "./notification-types";

function useManagement() {
  const session = useNotificationSession();
  const catalog = useNotificationCatalog();
  const rules = useNotificationRules();
  const destinations = useNotificationDestinations();
  const members = useNotificationMembers();
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: session.key });
  const client = () => getUnkeyClient().notifications;
  async function cacheItem<T extends { id: string }>(
    collection: "rules" | "destinations",
    item: T,
  ) {
    const queryKey = [...session.key, collection];
    await queryClient.cancelQueries({ queryKey });
    queryClient.setQueryData<T[]>(queryKey, (items = []) =>
      items.some((current) => current.id === item.id)
        ? items.map((current) => (current.id === item.id ? item : current))
        : [...items, item],
    );
  }

  async function removeItem(collection: "rules" | "destinations", id: string) {
    const queryKey = [...session.key, collection];
    await queryClient.cancelQueries({ queryKey });
    queryClient.setQueryData<Array<{ id: string }>>(queryKey, (items) =>
      items?.filter((item) => item.id !== id),
    );
  }

  return {
    rules: rules.data?.map(toRuleModel) ?? [],
    destinations: destinations.data ?? [],
    members: members.data ?? [],
    types: supportedNotificationTypes(catalog.data?.eventTypes ?? []),
    channels: catalog.data?.channels ?? [],
    audiences: catalog.data?.audiences ?? [],
    currentUserId: session.user?.id ?? null,
    canManage: session.canManage,
    ready: Boolean(catalog.data && rules.data && destinations.data && members.data),
    error: catalog.error ?? rules.error ?? destinations.error ?? members.error,
    retry: invalidate,
    actions: {
      async createRule(rule: Omit<Rule, "id">, idempotencyKey: string) {
        const { data } = await client().createRule({ ...ruleFields(rule), idempotencyKey });
        await cacheItem("rules", data);
        await invalidate();
      },
      async updateRule(rule: Rule) {
        const { data } = await client().updateRule({ ruleId: rule.id, ...ruleFields(rule) });
        await cacheItem("rules", data);
        await invalidate();
      },
      async deleteRule(ruleId: string) {
        await client().deleteRule({ ruleId });
        await removeItem("rules", ruleId);
        await invalidate();
      },
      async createDestination(input: Parameters<Unkey["notifications"]["createDestination"]>[0]) {
        const { data } = await client().createDestination(input);
        await cacheItem("destinations", data);
        await invalidate();
        return data;
      },
      async updateDestination(input: Parameters<Unkey["notifications"]["updateDestination"]>[0]) {
        const { data } = await client().updateDestination(input);
        await cacheItem("destinations", data);
        await invalidate();
        return data;
      },
      async deleteDestination(destinationId: string) {
        await client().deleteDestination({ destinationId });
        await removeItem("destinations", destinationId);
        await invalidate();
      },
    },
  };
}

const NotificationContext = createContext<ReturnType<typeof useManagement> | null>(null);

export function NotificationProvider({ children }: { children: ReactNode }) {
  const session = useNotificationSession();
  if (session.isLoading) {
    return <p className="p-6 text-sm text-gray-11">Loading notifications…</p>;
  }
  if (!session.canManage) {
    return (
      <p className="p-6 text-sm text-gray-11">
        Only workspace admins can manage notification routes.
      </p>
    );
  }
  return (
    <NotificationManagement key={`${session.workspace?.id}:${session.user?.id}`}>
      {children}
    </NotificationManagement>
  );
}

function NotificationManagement({ children }: { children: ReactNode }) {
  const state = useManagement();
  if (!state.ready) {
    return (
      <div className="space-y-3 p-6 text-sm text-gray-11">
        <p role={state.error ? "alert" : "status"}>
          {state.error
            ? getErrorMessage(state.error, "Couldn't load notifications.")
            : "Loading notifications…"}
        </p>
        {state.error ? (
          <Button variant="outline" onClick={() => state.retry()}>
            Retry
          </Button>
        ) : null}
      </div>
    );
  }
  return (
    <NotificationContext.Provider value={state}>
      {state.error ? (
        <div role="alert" className="flex items-center gap-3 p-6 text-sm text-error-11">
          Couldn't refresh notifications. Some details may be out of date.
          <Button variant="outline" size="sm" onClick={() => state.retry()}>
            Retry
          </Button>
        </div>
      ) : null}
      {children}
    </NotificationContext.Provider>
  );
}

export function useNotifications() {
  const context = useContext(NotificationContext);
  if (!context) {
    throw new Error("useNotifications must be used within NotificationProvider");
  }
  return context;
}
