import { allPolicies } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import { notFound } from "next/navigation";
export const generateStaticParams = async () =>
allPolicies.map((policy) => ({ slug: policy._raw.flattenedPath }));

export const generateMetadata = ({ params }: { params: { slug: string } }) => {
  const policy = allPolicies.find((policy) => policy._raw.flattenedPath === `policies/${params.slug}`);
  if (!policy) {
    return notFound();
  }
  return {
    title: policy.title,
    description: policy.title,
    openGraph: {
      title: policy.title,
      description: policy.title,
      type: "article",
      image: `https://unkey.dev/og?title=${encodeURIComponent(policy.title)}`,
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

const PolicyLayout = ({ params }: { params: { slug: string } }) => {
  const policy = allPolicies.find((policy) => policy._raw.flattenedPath === `policies/${params.slug}`);
  if (!policy) {
    return notFound();
  }
  const Content = getMDXComponent(policy.body.code);

  return (
    <>
      <article className="w-full max-w-3xl p-4 mx-auto prose lg:prose-md">
        <div className="max-w-2xl py-8 mx-auto mb-8 ">
          <h1 className="text-center">{policy.title}</h1>
        </div>
        <Content />
      </article>
    </>
  );
};

export default PolicyLayout;
