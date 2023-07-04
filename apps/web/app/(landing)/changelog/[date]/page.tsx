import { allChangelogs } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import { notFound } from "next/navigation";
import { date } from "zod";

export const generateStaticParams = async () =>
  allChangelogs.map((c) => ({
    date: new Date(c.date).toISOString().split("T")[0],
  }));

export const generateMetadata = ({ params }: { params: { date: string } }) => {
  const changelog = allChangelogs.find(
    (c) => new Date(c.date).toISOString().split("T")[0] === params.date
  );
  if (!changelog) {
    return notFound();
  }
  return {
    title: changelog.title,
    description: `changelog for ${date}`,
    openGraph: {
      title: changelog.title,
      description: `changelog for ${date}`,
      type: "article",
    },
  };
};

export default function ChangelogPage({
  params,
}: {
  params: { date: string };
}) {
  const changelog = allChangelogs.find(
    (c) => new Date(c.date).toISOString().split("T")[0] === params.date
  );
  if (!changelog) {
    return notFound();
  }

  const Content = getMDXComponent(changelog.body.code);

  return (
    <article className="prose lg:prose-lg w-full max-w-4xl mx-auto p-4">
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
