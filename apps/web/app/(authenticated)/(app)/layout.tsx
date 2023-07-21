import { Toaster } from "@/components/ui/toaster";
import { ReactQueryProvider } from "@/components/dashboard/react-query-provider";

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <ReactQueryProvider>
      {children}
      <Toaster />
    </ReactQueryProvider>
  );
}
