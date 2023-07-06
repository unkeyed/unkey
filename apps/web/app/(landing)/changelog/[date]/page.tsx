import { allChangelogs } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import { notFound } from "next/navigation";

export const generateStaticParams = async () =>
  allChangelogs.map((c) => ({
    date: new Date(c.date).toISOString().split("T")[0],
  }));

export const generateMetadata = ({ params }: { params: { date: string } }) => {
  const changelog = allChangelogs.find(
    (c) => new Date(c.date).toISOString().split("T")[0] === params.date,
  );
  if (!changelog) {
    return notFound();
  }
  return {
    title: changelog.title,
    description: `changelog for ${changelog.date}`,
    image: `https://unkey.dev/og?title=${encodeURIComponent(changelog.title)}`,
    openGraph: {
      title: changelog.title,
      description: `changelog for ${changelog.date}`,
      type: "article",
      image: `https://unkey.dev/og?title=${encodeURIComponent(changelog.title)}`,
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

export default function ChangelogPage({
  params,
}: {
  params: { date: string };
}) {
  const changelog = allChangelogs.find(
    (c) => new Date(c.date).toISOString().split("T")[0] === params.date,
  );
  if (!changelog) {
    return notFound();
  }

  const Content = getMDXComponent(changelog.body.code);

  return (
    <article className="prose w-full max-w-3xl mx-auto p-4">
      <div className="mb-8 py-8 mx-auto max-w-2xl ">
        <h1 className="text-center">{changelog.title}</h1>
        <span className="flex justify-center text-sm text-gray-600">
          <time dateTime={changelog.date} className="mx-2 ">
            {new Date(changelog.date).toDateString()}
          </time>
        </span>
      </div>
      <Content />
    </article>
  );
}
