import { Container } from "@/components/landing/container";
import { allJobs } from "contentlayer/generated";
import { useMDXComponent } from "next-contentlayer/hooks";
import { redirect } from "next/navigation";
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

  return {
    title: `${job?.title} | Unkey`,
    description: job?.description,
    openGraph: {
      title: `${job?.title} | Unkey`,
      description: job?.description,
      url: `https://unkey.dev/careers/${job?.slug}`,
    },
    twitter: {
      card: "summary_large_image",
      title: `${job?.title} | Unkey`,
      description: job?.description,
      site: "@unkeydev",
      creator: "@unkeydev",
    },
    icons: {
      shortcut: "/images/landing/unkey.png",
    },
  };
}

export const generateStaticParams = async () =>
  allJobs.map((j) => ({
    slug: j.slug,
  }));

export default function JobPage({ params }: { params: { slug: string } }) {
  const job = allJobs.find((j) => j.slug === params.slug && j.visible);

  if (!job) {
    redirect("/careers");
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
      <div className="relative mt-16 flex flex-col items-start space-y-8 lg:mt-32 lg:flex-row lg:space-y-0 ">
        <div className="top-32 mx-auto h-max w-full self-start px-4 sm:px-6 lg:sticky lg:w-2/5 lg:px-8 ">
          <Link
            href="/careers"
            className="text-content-subtle hover:text-foreground flex items-center gap-1 text-xs duration-200"
          >
            <ArrowLeft className="h-4 w-4" /> Back to all careers
          </Link>
          <div className="mt-4 pb-10">
            <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-6xl">
              {job.title}
            </h2>
            <p className="mt-2 text-gray-500">{job.description}</p>
          </div>
          <div className="flex items-center justify-between gap-4">
            <Link
              target="_blank"
              className="flex w-full items-center justify-center rounded-md border border-gray-900 bg-gray-900 px-4 py-2 text-center text-sm font-medium text-gray-100 duration-150 hover:bg-gray-100 hover:text-gray-900"
              href={"mailto:jobs@unkey.dev"}
            >
              Apply
            </Link>
          </div>

          <dl className="mt-10 grid grid-cols-2 gap-6">
            {Object.entries(perks).map(([label, Icon]) => (
              <div key={label} className="flex items-center gap-2">
                <dd className="text-sm text-gray-400">{<Icon className="h-4 w-4" />}</dd>
                <dt className="text-sm font-medium text-gray-900">{label}</dt>
              </div>
            ))}
          </dl>
        </div>

        <div className="prose lg:prose-md w-full border-gray-100 lg:w-3/5 lg:border-l lg:pl-8">
          <Content />
        </div>
      </div>
    </Container>
  );
}
