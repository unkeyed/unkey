import { CommandMenu } from "@/components/dashboard/command-menu";
import { Toaster } from "@/components/ui/toaster";
import { ReactQueryProvider } from "./react-query-provider";
import { ThemeProvider } from "./theme-provider";
import { auth } from "@clerk/nextjs";
import PostHogClient from "@/lib/posthog";

export default function Layout({ children }: { children: React.ReactNode }) {
  const { userId } = auth();
  if (userId) {
    const postHog = PostHogClient();
    postHog.identify({
      distinctId: userId,
    });
  }
  return (
    <ReactQueryProvider>
      <ThemeProvider attribute="class">
        {children}
        <Toaster />
        <CommandMenu />
      </ThemeProvider>
    </ReactQueryProvider>
  );
}
