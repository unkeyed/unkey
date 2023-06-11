import { ClerkProvider, SignIn, SignedIn, SignedOut } from "@clerk/nextjs";
import { Particles } from "@/components/particles";

export default function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ClerkProvider
      signInUrl="/auth/sign-in"
      signUpUrl="/auth/sign-up"
      appearance={{
        variables: {
          colorPrimary: "#5C36A3",
          colorText: "#5C36A3",
        },
      }}
      afterSignInUrl={"/overview"}
      afterSignUpUrl={"/onboarding"}
    >
      <SignedIn>{children}</SignedIn>
      <SignedOut>
        <div className="flex items-center justify-center w-screen h-screen  bg-gradient-to-t from-transparent from-violet-400/0 to-violet-400/20">
          <Particles
            className="absolute inset-0 -z-10 "
            vy={-1}
            quantity={50}
            staticity={200}
            color="#7c3aed"
          />

          <SignIn />
        </div>
      </SignedOut>
    </ClerkProvider>
  );
}
