import {
  type Credits,
  type Database,
  type Key,
  and,
  createConnection,
  eq,
  schema,
  sql,
} from "@/pkg/db";
import { ConsoleLogger } from "@unkey/worker-logging";
import type { Env } from "../env";
import { limitRequestSchema, revalidateRequestSchema } from "./interface";

export class DurableObjectUsagelimiter implements DurableObject {
  private readonly state: DurableObjectState;
  private readonly db: Database;
  private lastRevalidate = 0;
  private key: Key | undefined = undefined;
  private credit: Credits | undefined = undefined;
  private readonly env: Env;

  constructor(state: DurableObjectState, env: Env) {
    this.state = state;
    this.env = env;
    this.db = createConnection({
      host: env.DATABASE_HOST,
      password: env.DATABASE_PASSWORD,
      username: env.DATABASE_USERNAME,
      retry: false,
      logger: new ConsoleLogger({
        requestId: "",
        application: "api",
        environment: this.env.ENVIRONMENT,
      }),
    });
  }

  async fetch(request: Request) {
    const logger = new ConsoleLogger({
      application: "api",
      environment: this.env.ENVIRONMENT,
      requestId: request.headers.get("Unkey-Request-Id") ?? "",
      defaultFields: {
        durableObjectId: this.state.id.toString(),
        durableObjectClass: "DurableObjectUsagelimiter",
        environment: this.env.ENVIRONMENT,
      },
    });

    const url = new URL(request.url);
    switch (url.pathname) {
      case "/revalidate": {
        const req = revalidateRequestSchema.parse(await request.json());
        const creditId = req.creditId;
        const keyId = req.keyId;

        if (creditId) {
          // New credits system
          this.credit = await this.db.query.credits.findFirst({
            where: (table, { eq }) => eq(table.id, creditId),
          });
        } else if (keyId) {
          // Legacy key-based system
          this.key = await this.db.query.keys.findFirst({
            where: (table, { and, eq, isNull }) =>
              and(eq(table.id, keyId), isNull(table.deletedAtM)),
          });
        }
        this.lastRevalidate = Date.now();
        return Response.json({});
      }
      case "/limit": {
        const req = limitRequestSchema.parse(await request.json());

        // Determine if we're using new credits system or legacy key-based system
        const creditId = req.creditId;
        const keyId = req.keyId;

        if (creditId) {
          // New credits table system
          if (!this.credit) {
            this.credit = await this.db.query.credits.findFirst({
              where: (table, { eq }) => eq(table.id, creditId),
            });
            this.lastRevalidate = Date.now();
          }

          if (!this.credit) {
            logger.error("credit not found", { creditId });
            return Response.json({
              valid: false,
            });
          }

          const currentRemaining = this.credit.remaining;

          if (currentRemaining < req.cost && req.cost !== 0) {
            return Response.json({
              valid: false,
              remaining: Math.max(0, currentRemaining),
            });
          }

          if (req.cost !== 0) {
            this.credit.remaining = Math.max(0, currentRemaining - req.cost);

            this.state.waitUntil(
              this.db
                .update(schema.credits)
                .set({ remaining: sql`${schema.credits.remaining}-${req.cost}` })
                .where(
                  and(
                    eq(schema.credits.id, this.credit.id),
                    sql`${schema.credits.remaining} >= ${req.cost}`, // ensure sufficient credits
                  ),
                )
                .execute(),
            );
          }

          // revalidate every minute
          if (Date.now() - this.lastRevalidate > 60_000) {
            logger.info("revalidating in the background", {
              creditId: this.credit.id,
            });

            this.state.waitUntil(
              this.db.query.credits
                .findFirst({
                  where: (table, { eq }) => eq(table.id, creditId),
                })
                .execute()
                .then((credit) => {
                  this.credit = credit;
                  this.lastRevalidate = Date.now();
                }),
            );
          }

          return Response.json({
            valid: true,
            remaining: this.credit.remaining,
          });
        }

        if (!keyId) {
          logger.error("Neither creditId nor keyId provided");
          return Response.json({
            valid: false,
          });
        }

        // Legacy key-based system
        if (!this.key) {
          this.key = await this.db.query.keys.findFirst({
            where: (table, { and, eq, isNull }) =>
              and(eq(table.id, keyId), isNull(table.deletedAtM)),
          });
          this.lastRevalidate = Date.now();
        }

        if (!this.key) {
          logger.error("key not found", { keyId });
          return Response.json({
            valid: false,
          });
        }

        if (this.key.remaining === null) {
          logger.warn("key does not have remaining requests enabled", {
            key: this.key,
          });
          return Response.json({
            valid: true,
          });
        }

        const currentRemaining = this.key.remaining;

        if (currentRemaining < req.cost && req.cost !== 0) {
          return Response.json({
            valid: false,
            remaining: Math.max(0, currentRemaining),
          });
        }

        if (req.cost !== 0) {
          this.key.remaining = Math.max(0, currentRemaining - req.cost);

          this.state.waitUntil(
            this.db
              .update(schema.keys)
              .set({ remaining: sql`${schema.keys.remaining}-${req.cost}` })
              .where(
                and(
                  eq(schema.keys.id, this.key.id),
                  sql`${schema.keys.remaining} >= ${req.cost}`, // ensure sufficient credits
                ),
              )
              .execute(),
          );
        }

        // revalidate every minute
        if (Date.now() - this.lastRevalidate > 60_000) {
          logger.info("revalidating in the background", {
            keyId: this.key.id,
          });

          this.state.waitUntil(
            this.db.query.keys
              .findFirst({
                where: (table, { and, eq, isNull }) =>
                  and(eq(table.id, keyId), isNull(table.deletedAtM)),
              })
              .execute()
              .then((key) => {
                this.key = key;
                this.lastRevalidate = Date.now();
              }),
          );
        }

        return Response.json({
          valid: true,
          remaining: this.key.remaining,
        });
      }
    }
    return new Response("invalid path", { status: 404 });
  }
}
