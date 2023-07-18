import { env } from "@/env.mjs";
import { prismaClient } from "@/lib/prisma";

import { Unkey } from "@unkey/api";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
const unkey = new Unkey({ token: env.UNKEY_TOKEN });
export async function GET(request: Request) {
  //   const existingUser = await prismaClient.user.findFirst({
  //     where: {
  //       email: user!.emailAddresses[0].emailAddress,
  //     },
  //   });
  //   if (!existingUser) {
  //     await prismaClient.user.create({
  //       data: {
  //         email: user!.emailAddresses[0].emailAddress,
  //         firstName: user?.firstName,
  //         lastName: user?.lastName,
  //       },
  //     });
  //   }

  // create key

  const created = await unkey.keys.create({
    apiId: env.UNKEY_API_ID,
    prefix: "glam",
    byteLength: 16,
    ownerId: "glamboyosa",
    meta: {
      hello: "human",
    },
    remaining: 2,
  });

  console.log(created.key);

  cookies().set({
    name: "unkey-limited-key",
    value: created.key,
    secure: true,
  });

  redirect("/");
}
export const runtime = "edge";
