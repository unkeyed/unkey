import { Container } from "@/components/landing/container";
import { allJobs } from "contentlayer/generated";
import { useMDXComponent } from "next-contentlayer/hooks";
import { notFound } from "next/navigation";
import React from "react";

import { ArrowLeft, Banknote, BarChart, Cake, Globe, LucideIcon } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

type Props = {
  params: { slug: string };
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  // read route params
  const job = allJobs.find((j) => j.slug === params.slug && j.visible);
  const baseUrl = process.env.VERCEL_URL
    ? `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";

  return {
    title: `${job?.title} | Unkey`,
    description: job?.description,
    openGraph: {
      title: `${job?.title} | Unkey`,
      description: job?.description,
      url: `${baseUrl}/careers/${job?.slug}`,
      siteName: "unkey.dev",
      images: [
        {
          url: `${baseUrl}/og/changelog?title=${job?.title}`,
          width: 1200,
          height: 675,
        },
      ],
    },
    twitter: {
      title: `${job?.title} | Unkey`,
      card: "summary_large_image",
    },
    icons: {
      shortcut: "/unkey.png",
    },
  };
}

export const generateStaticParams = async () =>
  allJobs.map((j) => ({
    slug: j.slug,
  }));

export default function JobPage({
  params,
}: {
  params: { slug: string };
}) {
  const job = allJobs.find((j) => j.slug === params.slug && j.visible);

  if (!job) {
    return notFound();
  }

  const perks: Record<string, LucideIcon> = {
    "Remote Anywhere": Globe,
    [job.salary]: Banknote,
    "Stock Options": BarChart,
    "Unlimited PTO": Cake,
  };
  const Content = useMDXComponent(job.body.code);

  return (
    <Container>
      <div className="relative flex flex-col items-start mt-16 space-y-8 lg:flex-row lg:mt-32 lg:space-y-0 ">
        <div className="self-start w-full px-4 mx-auto lg:sticky top-32 h-max lg:w-2/5 sm:px-6 lg:px-8 ">
          <Link
            href="/templates"
            className="flex items-center gap-1 text-xs duration-200 text-content-subtle hover:text-foreground"
          >
            <ArrowLeft className="w-4 h-4" /> Back to all careers
          </Link>
          <div className="pb-10 mt-4">
            <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-6xl">
              {job.title}
            </h2>
            <p className="mt-2 text-gray-500">{job.description}</p>
          </div>
          <div className="flex items-center justify-between gap-4">
            <Link
              target="_blank"
              className="flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-center text-gray-100 bg-gray-900 duration-150 border rounded-md border-gray-900 hover:bg-gray-100 hover:text-gray-900"
              href={"mailto:jobs@unkey.dev"}
            >
              Apply
            </Link>
          </div>

          <dl className="grid grid-cols-2 gap-6 mt-10">
            {Object.entries(perks).map(([label, Icon]) => (
              <div key={label} className="flex items-center gap-2">
                <dd className="text-sm text-gray-400">{<Icon className="w-4 h-4" />}</dd>
                <dt className="text-sm font-medium text-gray-900">{label}</dt>
              </div>
            ))}
          </dl>
        </div>

        <div className="w-full border-gray-100 lg:border-l lg:pl-8 lg:w-3/5 prose lg:prose-md">
          <Content />
        </div>
      </div>
    </Container>
  );
}
