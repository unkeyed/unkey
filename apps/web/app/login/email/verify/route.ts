import { lucia } from "@/lib/auth/index";
import { db, eq, schema } from "@/lib/db";
import type { User } from "lucia";
import { isWithinExpirationDate } from "oslo";

export const GET = async (request: Request) => {
  const url = new URL(request.url);
  const email = url.searchParams.get("email");
  const otp = url.searchParams.get("otp");

  if (typeof otp !== "string") {
    return new Response(null, {
      status: 400,
    });
  }

  const user = await db.query.users.findFirst({
    where: (table, { eq }) => eq(table.email, email!),
  });
  console.log({ user });

  if (!user) {
    return new Response(null, {
      status: 403,
    });
  }

  const validCode = await verifyVerificationCode(user, otp);
  if (!validCode) {
    return new Response(null, {
      status: 400,
    });
  }

  await lucia.invalidateUserSessions(user.id);

  const session = await lucia.createSession(user.id, {});
  const sessionCookie = lucia.createSessionCookie(session.id);
  return new Response(null, {
    status: 302,
    headers: {
      Location: "/",
      "Set-Cookie": sessionCookie.serialize(),
    },
  });
};

async function verifyVerificationCode(
  user: { id: string; email: string },
  code: string,
): Promise<boolean> {
  const dbCode = await db.query.otps.findFirst({
    where: (table, { eq, and }) => and(eq(table.userId, user.id), eq(table.otp, code)),
  });

  if (!dbCode) {
    return false;
  }
  await db.delete(schema.otps).where(eq(schema.otps.id, dbCode.id));

  if (!isWithinExpirationDate(dbCode.expiresAt)) {
    return false;
  }
  if (dbCode.email !== user.email) {
    return false;
  }
  return true;
}
