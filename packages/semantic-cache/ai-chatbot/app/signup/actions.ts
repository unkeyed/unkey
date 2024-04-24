"use server";

import { signIn } from "@/auth";
import { ResultCode, getStringFromBuffer } from "@/lib/utils";
import { kv } from "@vercel/kv";
import { AuthError } from "next-auth";
import { z } from "zod";
import { getUser } from "../login/actions";

export async function createUser(email: string, hashedPassword: string, salt: string) {
  const existingUser = await getUser(email);

  if (existingUser) {
    return {
      type: "error",
      resultCode: ResultCode.UserAlreadyExists,
    };
  }
  const user = {
    id: crypto.randomUUID(),
    email,
    password: hashedPassword,
    salt,
  };

  await kv.hmset(`user:${email}`, user);

  return {
    type: "success",
    resultCode: ResultCode.UserCreated,
  };
}

interface Result {
  type: string;
  resultCode: ResultCode;
}

export async function signup(
  _prevState: Result | undefined,
  formData: FormData,
): Promise<Result | undefined> {
  const email = formData.get("email") as string;
  const password = formData.get("password") as string;

  const parsedCredentials = z
    .object({
      email: z.string().email(),
      password: z.string().min(6),
    })
    .safeParse({
      email,
      password,
    });

  if (parsedCredentials.success) {
    const salt = crypto.randomUUID();

    const encoder = new TextEncoder();
    const saltedPassword = encoder.encode(password + salt);
    const hashedPasswordBuffer = await crypto.subtle.digest("SHA-256", saltedPassword);
    const hashedPassword = getStringFromBuffer(hashedPasswordBuffer);

    try {
      const result = await createUser(email, hashedPassword, salt);

      if (result.resultCode === ResultCode.UserCreated) {
        await signIn("credentials", {
          email,
          password,
          redirect: false,
        });
      }

      return result;
    } catch (error) {
      if (error instanceof AuthError) {
        switch (error.type) {
          case "CredentialsSignin":
            return {
              type: "error",
              resultCode: ResultCode.InvalidCredentials,
            };
          default:
            return {
              type: "error",
              resultCode: ResultCode.UnknownError,
            };
        }
      }
      return {
        type: "error",
        resultCode: ResultCode.UnknownError,
      };
    }
  } else {
    return {
      type: "error",
      resultCode: ResultCode.InvalidCredentials,
    };
  }
}
