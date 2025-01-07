import { MDX } from "@/components/mdx-content";
import { allPolicies } from "content-collections";
import type { Policy } from "content-collections";
import { notFound } from "next/navigation";

export const generateStaticParams = async () =>
  allPolicies.map((policy) => ({
    slug: policy.slug,
  }));

export const generateMetadata = async ({
  params,
}: {
  params: { slug: string };
}) => {
  const policy = allPolicies.find((policy) => policy.slug === `${params.slug}`);
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
  };
};

const PolicyLayout = async ({ params }: { params: { slug: string } }) => {
  const policy = allPolicies.find((post) => post.slug === `${params.slug}`) as Policy;

  return (
    <>
      <article className="w-full max-w-3xl p-4 mx-auto prose-invert">
        <div className="max-w-2xl py-8 mx-auto mb-8 ">
          <h1 className="text-center text-white/90 text-4xl">{policy.title}</h1>
        </div>
        <MDX code={policy.mdx} />
      </article>
    </>
  );
};

export default PolicyLayout;
