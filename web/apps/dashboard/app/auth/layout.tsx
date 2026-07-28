import { FullScreenContent, FullScreenLayout, Logo } from "@unkey/ui";
import Link from "next/link";
import type React from "react";
import { AuthSwitchButton } from "./auth-switch-button";
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
    <FullScreenLayout className="overflow-x-hidden bg-gray-2 dark:bg-background">
      <nav className="flex items-center justify-between h-16 w-full shrink-0 px-6">
        <Link href="/">
          <Logo />
        </Link>
        <AuthSwitchButton />
      </nav>
      <FullScreenContent className="px-4 py-8">
        <div className="w-full max-w-[440px] bg-none px-6 py-10 sm:px-12">
          <div className="flex w-full flex-col gap-10">
            <RadarProvider>{children}</RadarProvider>
            <p className="text-xs text-center text-gray-9 text-balance">
              By continuing, you agree to Unkey's{" "}
              <Link
                className="underline hover:text-gray-11"
                href="https://www.unkey.com/policies/terms"
                target="_blank"
                rel="noopener noreferrer"
              >
                Terms of Service
              </Link>{" "}
              and{" "}
              <Link
                className="underline hover:text-gray-11"
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
