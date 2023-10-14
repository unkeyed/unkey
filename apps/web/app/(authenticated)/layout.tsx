import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ClerkProvider, auth } from "@clerk/nextjs";
import * as Sentry from "@sentry/nextjs";
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
          afterSignUpUrl="/onboarding"
          appearance={{
            variables: {
              colorPrimary: "#5C36A3",
              colorText: "#5C36A3",
            },
          }}
        >
          <SetUserInSentry />
          {children}
        </ClerkProvider>
      </TooltipProvider>
      <Toaster />
    </>
  );
}

const SetUserInSentry: React.FC = () => {
  const { userId } = auth();
  if (!userId) {
    Sentry.setUser(null);
  } else {
    Sentry.setUser({
      id: userId,
    });
  }
  return null;
};
