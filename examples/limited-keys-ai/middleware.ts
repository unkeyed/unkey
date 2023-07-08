import { authMiddleware } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { redirectToSignIn } from "@clerk/nextjs";
import { Unkey } from "@unkey/api";
import { env } from "@/env.mjs";
import { RequestCookie } from "next/dist/compiled/@edge-runtime/cookies";
import { prismaClient } from "./lib/prisma";
const unkey = new Unkey({ token: env.UNKEY_TOKEN });
export default authMiddleware({
  async beforeAuth(req, _evt) {
    if (!req.cookies.has("unkey-limited-key")) {
      return NextResponse.next();
    }
    // get cookie and verify it
    const { value: key } = req.cookies.get(
      "unkey-limited-key"
    ) as RequestCookie;

    const res = await unkey.keys.verify({
      key,
    });

    if (!res.valid) {
      return new NextResponse(
        JSON.stringify({ success: false, message: "authentication failed" }),
        { status: 401, headers: { "content-type": "application/json" } }
      );
    }
    return NextResponse.next();
  },
  async afterAuth(auth, req, _evt) {
    if (!auth.userId && !auth.isPublicRoute) {
      return redirectToSignIn({ returnBackUrl: "/" });
    }
    if (auth.userId && !auth.orgId) {
      // save user
      const existingUser = await prismaClient.user.findFirst({
        where: {
          email: auth.user!.emailAddresses[0].emailAddress,
        },
      });
      if (!existingUser) {
        await prismaClient.user.create({
          data: {
            email: auth.user!.emailAddresses[0].emailAddress,
            firstName: auth.user?.firstName,
            lastName: auth.user?.lastName,
          },
        });
      }
      // Create a new expiry date by adding 7 days to the current date
      const currentDate = new Date();

      const expiry = new Date();
      expiry.setDate(currentDate.getDate() + 7);

      // create key
      // TO-DO: Add Limit
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
      if (req.cookies.has("unkey-limited-key")) {
        return NextResponse.redirect("/");
      }
      cookies().set({
        name: "unkey-limited-key",
        value: created.key,
      });

      return NextResponse.redirect("/");
    }
  },
});

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
