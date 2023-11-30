import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ClerkProvider } from "@clerk/nextjs";

export const dynamic = "force-dynamic";

export default function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <TooltipProvider>
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
      </TooltipProvider>
      <Toaster />
    </>
  );
}
