import { github } from "@/lib/auth/index";
import { db, schema } from "@/lib/db";
import { newId } from "@unkey/id";
// app/login/github/route.ts
import { generateState } from "arctic";
import { cookies } from "next/headers";
import Link from "next/link";
import { alphabet, generateRandomString } from "oslo/crypto";

export default async function Page() {
  const email = "andreas@unkey.dev";

  const otp = await generateEmailVerificationCode("u_2KBHASUesSga4E5fimbUXuePn6c", email);

  return (
    <div>
      Your OTP: <pre>{otp}</pre>
      <Link href={`/login/email/verify?email=${email}&otp=${otp}`}>Link</Link>
    </div>
  );
}

async function generateEmailVerificationCode(userId: string, email: string): Promise<string> {
  const otp = generateRandomString(6, alphabet("0-9"));
  await db.insert(schema.otps).values({
    id: Math.random().toString(),
    userId,
    email,
    otp,
    expiresAt: new Date(Date.now() + 15 * 60 * 1000),
  });
  return otp;
}
