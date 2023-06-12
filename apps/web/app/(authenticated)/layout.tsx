import { ClerkProvider, SignedIn, SignedOut, RedirectToSignIn } from "@clerk/nextjs";
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
      <SignedIn>{children}</SignedIn>
      <SignedOut>
        <RedirectToSignIn afterSignInUrl="/app" afterSignUpUrl="/onboarding" />
      </SignedOut>
    </ClerkProvider>
  );
}
