"use client";

import { Button } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { usePathname } from "next/navigation";

export function AuthSwitchButton() {
  const pathname = usePathname();
  const isSignUp = pathname?.startsWith("/auth/sign-up") ?? false;

  const label = isSignUp ? "Log in" : "Sign up";
  const href = (isSignUp ? "/auth/sign-in" : "/auth/sign-up") as Route;

  return (
    <Button variant="outline" size="md" render={<Link href={href} className="font-medium" />}>
      {label}
    </Button>
  );
}
