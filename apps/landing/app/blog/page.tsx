import { Alert, AlertDescription } from "@/components/ui/alert/alert";

export const metadata = {
  title: "Blog | Unkey",
  description: "Latest blog posts and news from the Unkey team.",
  openGraph: {
    title: "Blog | Unkey",
    description: "Latest blog posts and news from the Unkey team.",
    url: "https://unkey.dev/blog",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/images/landing/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Blog | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
};

export default async function Blog() {
  return (
    <>
      <div className="bg-black">
        <div className="max-w-[1000px] mx-auto text-gray-100 text-center min-h-screen p-6">
          <p>Demo Page</p>
          <p>Shadcn Alert Example</p>
          <Alert variant="info" className="m-4">
            <AlertDescription variant="info">
              We provide a white-glove migration service as part of our startup plan. Interested?
              Request it here
            </AlertDescription>
          </Alert>
          <Alert variant="success" className="m-4">
            <AlertDescription variant="success">
              We provide a white-glove migration service as part of our startup plan. Interested?
              Request it here
            </AlertDescription>
          </Alert>
          <Alert variant="alert" className="m-4">
            <AlertDescription variant="alert">
              We provide a white-glove migration service as part of our startup plan. Interested?
              Request it here
            </AlertDescription>
          </Alert>
          <Alert variant="warning" className="m-4">
            <AlertDescription variant="warning">
              We provide a white-glove migration service as part of our startup plan. Interested?
              Request it here
            </AlertDescription>
          </Alert>
          <Alert variant="error" className="m-4">
            <AlertDescription variant="error">
              We provide a white-glove migration service as part of our startup plan. Interested?
              Request it here
            </AlertDescription>
          </Alert>
        </div>
      </div>
    </>
  );
}
