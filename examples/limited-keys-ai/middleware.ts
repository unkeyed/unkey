import { authMiddleware } from "@clerk/nextjs/server";
import { NextResponse, NextRequest } from "next/server";
import { cookies } from "next/headers";
import { Unkey } from "@unkey/api";
import { env } from "@/env.mjs";
const unkey = new Unkey({ token: env.UNKEY_TOKEN });
export default authMiddleware({
  async afterAuth(auth, req, evt) {
    if (auth.userId && !auth.orgId) {
      const currentDate = new Date();

      // Create a new date by adding 7 days to the current date
      const expiry = new Date();
      expiry.setDate(currentDate.getDate() + 7);

      // create key
      const created = await unkey.keys.create({
        apiId: env.UNKEY_API_ID,
        prefix: "glam",
        byteLength: 16,
        ownerId: "glamboyosa",
        meta: {
          hello: "human",
        },
        expires: expiry.getMilliseconds(),

        ratelimit: {
          type: "fast",
          limit: 10,
          refillRate: 1,
          refillInterval: 1000,
        },
      });

      console.log(created.key);

      cookies().set({
        name: "unkey-limited-key",
        value: created.key,
        secure: true,
      });
      return NextResponse.redirect("/");
    }
  },
});

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
