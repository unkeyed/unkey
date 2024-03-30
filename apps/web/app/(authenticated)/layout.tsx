import { ClerkProvider } from "@clerk/nextjs";
import type React from "react";

export const dynamic = "force-dynamic";

export default async function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ClerkProvider
      afterSignInUrl="/app"
      afterSignUpUrl="/new"
      appearance={{
        variables: {
          colorPrimary: "#5C36A3",
          colorText: "#5C36A3",
        },
      }}
    >
      {children}
    </ClerkProvider>
  );
}
