import { ClerkProvider, SignIn, SignedIn, SignedOut } from "@clerk/nextjs";
import { Particles } from "@/components/particles";
import { Toaster } from "sonner";
export default function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ClerkProvider
      signInUrl="/sign-in"
      signUpUrl="/sign-up"
      afterSignInUrl="/app"
      afterSignUpUrl="/app"
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
