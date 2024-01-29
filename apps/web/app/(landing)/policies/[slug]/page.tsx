import { MdxContent } from "@/components/landing/mdx-content";
import { POLICY_PATH, getFilePaths, getPolicy } from "@/lib/mdx-helper";
import { notFound } from "next/navigation";

export const generateStaticParams = async () => {
  const policies = await getFilePaths(POLICY_PATH);
  // Remove file extensions for page paths
  policies.map((path) => path.replace(/\.mdx?$/, "")).map((slug) => ({ params: { slug } }));
  return policies;
};

export const generateMetadata = async ({ params }: { params: { slug: string } }) => {
  const { frontmatter } = await getPolicy(params.slug);
  if (!frontmatter) {
    return notFound();
  }
  return {
    title: frontmatter.title,
    description: frontmatter.title,
    openGraph: {
      title: frontmatter.title,
      description: frontmatter.title,
      type: "article",
      image: `https://unkey.dev/og?title=${encodeURIComponent(frontmatter.title)}`,
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
  const { frontmatter, serialized } = await getPolicy(params.slug);
  if (!serialized) {
    return notFound();
  }

  return (
    <>
      <article className="w-full max-w-3xl p-4 mx-auto prose lg:prose-md">
        <div className="max-w-2xl py-8 mx-auto mb-8 ">
          <h1 className="text-center">{frontmatter.title}</h1>
        </div>
        <MdxContent source={serialized} />
      </article>
    </>
  );
};

export default PolicyLayout;
