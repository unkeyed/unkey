"use server";

import { cookies } from "next/headers";

export async function setCookie(name: string, value: string) {
  cookies().set({
    name,
    value,
    httpOnly: true,
  });
}

export async function getCookie(name: string) {
  const cookie = await cookies().get(name);
  return cookie?.value;
}
