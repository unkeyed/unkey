import { ReactQueryProvider } from "./ReactQueryProvider";
import { Toaster } from "@/components/ui/toaster";

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
