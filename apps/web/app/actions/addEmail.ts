"use server";

import { env } from "@/lib/env";
import { Resend } from "resend";
export async function addEmail(formData: FormData) {
  const { RESEND_API_KEY, RESEND_AUDIENCE_ID } = env();
  if (!RESEND_API_KEY || !RESEND_AUDIENCE_ID) {
    return { success: false };
  }

  const email = formData.get("email")?.toString();
  if (!email) {
    return { success: false };
  }
  const resend = new Resend(RESEND_API_KEY);

  await resend.contacts.create({
    audience_id: RESEND_AUDIENCE_ID,
    email: email,
  });

  return { success: true };
}
