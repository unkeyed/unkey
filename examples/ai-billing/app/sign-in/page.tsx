import { auth, signIn } from "@/auth";
import { redirect } from "next/navigation";
import { SignInButton } from "./button";

export default async function SignIn() {
  const session = await auth();
  if (session) {
    return redirect("/");
  }

  return (
    <form
      className="flex items-center justify-center w-full h-[600px]"
      action={async () => {
        "use server";
        await signIn("github");
      }}
    >
      <SignInButton />
    </form>
  );
}
