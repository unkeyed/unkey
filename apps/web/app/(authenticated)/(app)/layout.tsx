import { CommandMenu } from "@/components/dashboard/command-menu";
import { ReactQueryProvider } from "./react-query-provider";
import { ThemeProvider } from "./theme-provider";
export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <ReactQueryProvider>
      <ThemeProvider attribute="class">
        {children}
        <CommandMenu />
      </ThemeProvider>
    </ReactQueryProvider>
  );
}
