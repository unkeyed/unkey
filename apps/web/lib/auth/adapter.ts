import { and, eq, lt, schema } from "@unkey/db";
import { PlanetScaleDatabase } from "drizzle-orm/planetscale-serverless";
import type { Adapter, DatabaseSession, DatabaseUser } from "lucia";

export class LocalAdapter implements Adapter {
  private readonly db: PlanetScaleDatabase<typeof schema>;

  constructor(config: { db: PlanetScaleDatabase<typeof schema> }) {
    this.db = config.db;
  }

  public async deleteExpiredSessions(): Promise<void> {
    await this.db.delete(schema.sessions).where(lt(schema.sessions.expiresAt, new Date()));
  }
  public async deleteSession(sessionId: string): Promise<void> {
    await this.db.delete(schema.sessions).where(eq(schema.sessions.id, sessionId));
  }
  public async deleteUserSessions(userId: string): Promise<void> {
    await this.db.delete(schema.sessions).where(eq(schema.sessions.userId, userId));
  }
  public async getSessionAndUser(
    sessionId: string,
  ): Promise<[session: DatabaseSession | null, user: DatabaseUser | null]> {
    const session = await this.db.query.sessions.findFirst({
      where: (table, { and, eq, gt }) =>
        and(eq(table.id, sessionId), gt(table.expiresAt, new Date())),
    });
    if (!session) {
      console.log("no session found");
      return [null, null];
    }
    const user = await this.db.query.users.findFirst({
      where: (table, { eq }) => eq(table.id, session.userId),
    });
    if (!user) {
      console.log("no user found");
      return [null, null];
    }
    return [
      {
        id: session.id,
        userId: session.userId,
        expiresAt: session.expiresAt,
        attributes: {},
      },
      {
        id: user.id,
        attributes: {
          username: "anon",
        },
      },
    ];
  }
  public async getUserSessions(userId: string): Promise<DatabaseSession[]> {
    const sessions = await this.db.query.sessions.findMany({
      where: (table, { eq }) => eq(table.userId, userId),
    });
    return sessions.map((session) => ({
      id: session.id,
      userId: session.userId,
      expiresAt: session.expiresAt,
      attributes: {},
    }));
  }
  public async setSession(session: DatabaseSession): Promise<void> {
    await this.db.insert(schema.sessions).values({
      id: session.id,
      userId: session.userId,
      expiresAt: session.expiresAt,
    });
  }
  public async updateSessionExpiration(sessionId: string, expiresAt: Date): Promise<void> {
    await this.db
      .update(schema.sessions)
      .set({
        expiresAt,
      })
      .where(eq(schema.sessions.id, sessionId));
  }
}
