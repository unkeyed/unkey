import { TemplatesClient } from "./client";

export const metadata = {
  title: "Templates | Unkey",
  description: "Jumpstart your API development with our pre-built solutions.",
  openGraph: {
    title: "Templates | Unkey",
    description: "Jumpstart your API development with our pre-built solutions.",
    url: "https://unkey.com/templates",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.com/images/landing/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Templates | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
};

export default function TemplatesPage() {
  return (
    <div>
      <TemplatesClient />
    </div>
  );
}
