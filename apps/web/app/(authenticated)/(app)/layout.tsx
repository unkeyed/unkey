import { Toaster } from "@/components/ui/toaster";
import { ReactQueryProvider } from "./ReactQueryProvider";

export default function Layout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ReactQueryProvider>
      {children}
      <Toaster />
    </ReactQueryProvider>
  );
}
