import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { env } from "@/env.mjs";
import { prismaClient } from "@/lib/prisma";
import { Unkey } from "@unkey/api";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
const unkey = new Unkey({ token: env.UNKEY_TOKEN });
export default function Login() {
  async function loginHandler(data: FormData) {
    "use server";
    const email = data.getAll("email")[0] as string;
    const existingUser = await prismaClient.user.findFirst({
      where: {
        email,
      },
    });
    if (!existingUser) {
      await prismaClient.user.create({
        data: {
          email,
        },
      });
    }

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
    cookies().set({
      name: "unkey-limited-key",
      value: created.key,
      secure: false,
      httpOnly: false,
      path: "/",
    });
    console.log(created.key);

    return redirect("/");
  }
  return (
    <div className="flex w-full max-w-sm items-center space-x-2">
      <Input type="email" name="email" placeholder="Email" />
      <Button formAction={loginHandler} type="submit">
        Login
      </Button>
    </div>
  );
}
