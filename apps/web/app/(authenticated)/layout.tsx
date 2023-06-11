import { ClerkProvider, SignIn, SignedIn, SignedOut } from "@clerk/nextjs";
import { Particles } from "@/components/particles";

export default function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ClerkProvider
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
