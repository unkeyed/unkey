import { ToastProvider } from "@/components/ui/toast";

export default function Layout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <ToastProvider>{children}</ToastProvider>;
}
