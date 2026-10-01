import type { InfiniteData } from "@tanstack/react-query";
import type {
  V2PortalListSessionsResponseBody,
  V2PortalListSessionsResponseData,
} from "@unkey/api/models/components";

export type SessionGroup = V2PortalListSessionsResponseData;

export type SessionPage = {
  groups: SessionGroup[];
  cursor?: string;
};

export type SessionGroupSummary = {
  count: number;
  lastCreatedAt: number;
  lastExpiresAt: number;
};

export function toSessionPage(response: V2PortalListSessionsResponseBody): SessionPage {
  const { data, pagination } = response;
  if (pagination.hasMore && !pagination.cursor) {
    throw new Error("Portal sessions API returned a continuation page without a cursor");
  }
  return {
    groups: data,
    cursor: pagination.hasMore ? pagination.cursor : undefined,
  };
}

export function summarizeSessionGroup(group: SessionGroup): SessionGroupSummary {
  return group.sessions.reduce<SessionGroupSummary>(
    (summary, session) => ({
      count: summary.count + 1,
      lastCreatedAt: Math.max(summary.lastCreatedAt, session.createdAt),
      lastExpiresAt: Math.max(summary.lastExpiresAt, session.expiresAt),
    }),
    { count: 0, lastCreatedAt: 0, lastExpiresAt: 0 },
  );
}

export function sessionGroupScopes(group: SessionGroup): string[] {
  return [...new Set(group.sessions.flatMap((session) => session.scopes))].sort();
}

// Drops a revoked end user from cached pages. The caller updates the cache
// instead of refetching, because a revoke only changes this end user and an
// immediate refetch can hit a replica that still lists them.
export function removeSessionGroup(
  data: InfiniteData<SessionPage> | undefined,
  externalId: string,
): InfiniteData<SessionPage> | undefined {
  if (!data) {
    return data;
  }
  return {
    ...data,
    pages: data.pages.map((page) =>
      page.groups.some((group) => group.externalId === externalId)
        ? { ...page, groups: page.groups.filter((group) => group.externalId !== externalId) }
        : page,
    ),
  };
}
