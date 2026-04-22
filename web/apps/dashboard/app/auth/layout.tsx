import { getAuth } from "@/lib/auth/get-auth";
import { Page2 } from "@unkey/icons";
import { Logo } from "@unkey/ui";
import Link from "next/link";
import { redirect } from "next/navigation";
import type React from "react";

export const dynamic = "force-dynamic";

export default async function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { userId } = await getAuth(); // we want the one without redirect

  if (userId) {
    return redirect("/apis");
  }

  return (
    <div className="min-h-screen overflow-x-hidden bg-black">
      <nav className="container flex items-center justify-between h-16">
        <Link href="/">
          <Logo className="min-w-sm text-white" />
        </Link>
        <Link
          className="flex items-center h-8 gap-2 px-4 text-sm text-white duration-500 border rounded-lg bg-white/5 hover:bg-white hover:text-black border-white/10"
          href="https://www.unkey.com/docs"
          target="_blank"
        >
          <Page2 iconSize="md-thin" />
          Documentation
        </Link>
      </nav>
      <div className="flex min-h-screen pt-16 -mt-16">
        <div className="container relative flex flex-col items-center justify-center gap-8 lg:w-2/5">
          <div className="w-full max-w-sm">{children}</div>
          <div className="flex items-center justify-center ">
            <p className="p-4 text-xs text-center text-white/50 text-balance">
              By continuing, you agree to Unkey's{" "}
              <Link
                className="underline"
                href="https://www.unkey.com/policies/terms"
                target="_blank"
                rel="noopener noreferrer"
              >
                Terms of Service
              </Link>{" "}
              and{" "}
              <Link
                className="underline"
                href="https://www.unkey.com/policies/privacy"
                target="_blank"
                rel="noopener noreferrer"
              >
                Privacy Policy
              </Link>
              , and to receive periodic emails with updates.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
