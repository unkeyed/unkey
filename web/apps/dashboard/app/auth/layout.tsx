import { Page2 } from "@unkey/icons";
import { FullScreenContent, FullScreenLayout, Logo } from "@unkey/ui";
import Link from "next/link";
import type React from "react";
import { RadarProvider } from "./radar/radar-signals";

// NOTE: do not add a signed-in redirect here. Setting the session cookie in
// a server action re-renders this layout as part of the action response, so
// a redirect("/apis") from here races ahead of the action's own navigation
// (e.g. the invite flow's /join/success) and flashes the dashboard. The
// signed-in bounce lives in proxy.ts, where it only applies to document GETs.
export default function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <FullScreenLayout className="overflow-x-hidden bg-black">
      <nav className="container flex items-center justify-between h-16 w-full shrink-0">
        <Link href="/">
          <Logo className="md:min-w-sm text-white" />
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
      <FullScreenContent className="py-8">
        <div className="container relative flex flex-col items-center justify-center gap-8 lg:w-2/5">
          <div className="w-full max-w-sm">
            <RadarProvider>{children}</RadarProvider>
          </div>
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
      </FullScreenContent>
    </FullScreenLayout>
  );
}
