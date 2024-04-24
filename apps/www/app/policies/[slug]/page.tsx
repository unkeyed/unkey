import { type Policy, allPolicies } from "@/.contentlayer/generated";
import { MDX } from "@/components/mdx-content";
import { notFound } from "next/navigation";

export const generateStaticParams = async () =>
  allPolicies.map((policy) => ({ slug: policy._raw.flattenedPath.replace("policies/", "") }));

export const generateMetadata = async ({ params }: { params: { slug: string } }) => {
  const policy = allPolicies.find(
    (policy) => policy._raw.flattenedPath === `policies/${params.slug}`,
  );
  if (!policy) {
    notFound();
  }
  return {
    title: policy.title,
    description: policy.title,
    openGraph: {
      title: policy.title,
      description: policy.title,
      type: "article",
    },
    robots: {
      index: true,
      follow: true,
      nocache: true,
      googleBot: {
        index: true,
        follow: false,
        noimageindex: true,
        "max-video-preview": -1,
        "max-image-preview": "large",
        "max-snippet": -1,
      },
    },
  };
};

const PolicyLayout = async ({ params }: { params: { slug: string } }) => {
  const policy = allPolicies.find(
    (post) => post._raw.flattenedPath === `policies/${params.slug}`,
  ) as Policy;

  return (
    <>
      <article className="w-full max-w-3xl p-4 mx-auto prose-invert">
        <div className="max-w-2xl py-8 mx-auto mb-8 ">
          <h1 className="text-center text-white/90 text-4xl">{policy.title}</h1>
        </div>
        <MDX code={policy.body.code} />
      </article>
    </>
  );
};

export default PolicyLayout;
