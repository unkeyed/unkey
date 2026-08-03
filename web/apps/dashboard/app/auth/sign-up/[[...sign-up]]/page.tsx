import { env } from "@/lib/env";
import { redirect } from "next/navigation";
import { LocalSignUpPage } from "./local-page";

export default function SignUpPage() {
  if (env().AUTH_PROVIDER === "workos") {
    redirect("/auth/error?reason=entry");
  }

  return <LocalSignUpPage />;
}
