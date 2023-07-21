import { Toaster } from "@/components/ui/toaster";
import { ReactQueryProvider } from "@/components/dashboard/react-query-provider";
import { ThemeProvider } from "@/components/dashboard/theme-provider";

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <ReactQueryProvider>
      <ThemeProvider attribute="class">
        {children}
        <Toaster />
      </ThemeProvider>
    </ReactQueryProvider>
  );
}
