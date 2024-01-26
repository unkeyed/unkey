import { BlogHeading, BlogSubTitle, BlogTitle } from "@/components/blog-heading";
import { Frame } from "@/components/frame";
import { Alert, AlertDescription } from "@/components/ui/alert/alert";
import Image from "next/image";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/blog-table";

const data = [
  {
    Property: "Name",
    Description: "Full name of user",
    Color: "Gray",
  },
  {
    Property: "Age",
    Description: "Reported age",
    Color: "Black",
  },
  {
    Property: "Joined",
    Description: "Whether the user joined the community",
    Color: "White",
  },
];

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
        <div className="max-w-[880px] mx-auto text-center min-h-screen p-6">
          <h2 className="text-white text-left pl-24">Blog / Product</h2>
          <BlogHeading>
            <BlogTitle>How Unkey and extensions work</BlogTitle>
            <BlogSubTitle>
              Learn more about how we built the Unkey API and how it works under the hood.
            </BlogSubTitle>
          </BlogHeading>
          <Table className="w-[600px] mx-auto">
            <TableHeader>
              <TableRow>
                <TableHead>Property</TableHead>
                <TableHead>Description</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((data) => (
                <TableRow key={data.Property}>
                  <TableCell>{data.Property}</TableCell>
                  <TableCell>{data.Description}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <Frame className="mt-12 mb-32">
            <Image
              src={"/images/blog-images/funding/funding-cover.png"}
              alt={""}
              width={600}
              height={400}
            />
          </Frame>
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
