import { RootLayout } from "@/components/landing-components/root-layout";
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
    <html lang="en" className="h-full bg-neutral-950 text-base antialiased">
      <body className="flex min-h-full flex-col">
        <RootLayout>{children}</RootLayout>
        <Toaster />
      </body>
    </html>
  );
}
