import { env } from "@/lib/env";
import { redirect } from "next/navigation";
import { LocalSignInPage } from "./local-page";

export default function SignInPage() {
  if (env().AUTH_PROVIDER === "workos") {
    redirect("/auth/error?reason=entry");
  }

  return <LocalSignInPage />;
}
