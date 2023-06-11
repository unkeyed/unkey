import { ClerkProvider } from "@clerk/nextjs";
import { Particles } from "@/components/particles";
import { Toaster } from "sonner";
export default function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ClerkProvider
      signInUrl="/auth/sign-in"
      signUpUrl="/auth/sign-up"
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
  );
}
