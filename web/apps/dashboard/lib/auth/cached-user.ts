import { auth } from "@/lib/auth/server";
import type { User } from "@/lib/auth/types";

const userCache = new Map<string, { user: User | null; timestamp: number }>();
const USER_CACHE_TTL = 1000 * 60 * 5;
const USER_CACHE_MAX_ENTRIES = 5_000;

export async function getCachedUser(userId: string): Promise<User | null> {
  const cached = userCache.get(userId);
  if (cached && Date.now() - cached.timestamp < USER_CACHE_TTL) {
    return cached.user;
  }

  let user: User | null;
  try {
    user = await auth.getUser(userId);
  } catch {
    return null;
  }

  userCache.delete(userId);
  if (userCache.size >= USER_CACHE_MAX_ENTRIES) {
    const oldest = userCache.keys().next();
    if (!oldest.done) {
      userCache.delete(oldest.value);
    }
  }
  userCache.set(userId, { user, timestamp: Date.now() });
  return user;
}

export async function fetchUsersByActorIds(actorIds: string[]): Promise<Record<string, User>> {
  try {
    const unique = [...new Set(actorIds)];
    const users = await Promise.all(unique.map(getCachedUser));

    return users.reduce(
      (acc, user) => {
        if (user) {
          acc[user.id] = user;
        }
        return acc;
      },
      {} as Record<string, User>,
    );
  } catch (error) {
    console.error("Error fetching users:", error);
    return {};
  }
}
