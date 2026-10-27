import { getUnkeyClient } from "@/lib/unkey-client";
import type { NotificationPagination } from "@unkey/api/models/components";
import { UnkeyError } from "@unkey/api/models/errors";

export function creationRejected(error: unknown): boolean {
  return (
    error instanceof UnkeyError &&
    error.statusCode >= 400 &&
    error.statusCode < 500 &&
    error.statusCode !== 408 &&
    error.statusCode !== 409
  );
}

export async function notificationPages<T>(
  pages: AsyncIterable<{ result: { data: T[]; pagination: NotificationPagination } }>,
): Promise<T[]> {
  const items: T[] = [];
  const cursors = new Set<string>();
  for await (const { result } of pages) {
    items.push(...result.data);
    if (!result.pagination.hasMore) {
      break;
    }
    const cursor = result.pagination.nextCursor;
    if (!cursor) {
      throw new Error("Notifications API returned another page without a cursor");
    }
    if (cursors.has(cursor)) {
      throw new Error("Notifications API returned a repeated cursor");
    }
    cursors.add(cursor);
  }
  return items;
}

export async function listNotificationRules(signal?: AbortSignal) {
  return notificationPages(
    await getUnkeyClient().notifications.listRules({ limit: 100 }, { signal }),
  );
}

export async function listNotificationDestinations(signal?: AbortSignal) {
  return notificationPages(
    await getUnkeyClient().notifications.listDestinations({ limit: 100 }, { signal }),
  );
}

export async function listNotificationMembers(signal?: AbortSignal) {
  return notificationPages(
    await getUnkeyClient().notifications.listMembers({ limit: 100 }, { signal }),
  );
}
