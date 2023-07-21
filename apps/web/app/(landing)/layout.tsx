import { RootLayout } from "@/components/landing/root-layout";
import { Toaster } from "@/components/ui/toaster";

import "@/styles/tailwind/tailwind.css";

export const metadata = {
  title: {
    template: "%s - Unkey",
    default: "Unkey - API management made easy",
  },
};

export default function Layout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="h-full text-base antialiased bg-neutral-950">
      <body className="flex flex-col min-h-full">
        <RootLayout>{children}</RootLayout>
        <Toaster />
      </body>
    </html>
  );
}
