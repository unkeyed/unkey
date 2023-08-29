import { Toaster } from "@/components/ui/toaster";
import { ClerkProvider } from "@clerk/nextjs";

export default function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <ClerkProvider
        afterSignInUrl="/app"
        afterSignUpUrl="/onboarding"
        appearance={{
          variables: {
            colorPrimary: "#5C36A3",
            colorText: "#5C36A3",
          },
        }}
      >
        {children}
      </ClerkProvider>
      <Toaster />
    </>
  );
}
