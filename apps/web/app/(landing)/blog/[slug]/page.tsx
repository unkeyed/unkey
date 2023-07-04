import { allPosts } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import { notFound } from "next/navigation";

export const generateStaticParams = async () =>
  allPosts.map((post) => ({ slug: post._raw.flattenedPath }));

export const generateMetadata = ({ params }: { params: { slug: string } }) => {
  const post = allPosts.find((post) => post._raw.flattenedPath === `blog/${params.slug}`);
  if (!post) {
    return notFound();
  }
  return {
    title: post.title,
    description: post.excerpt,
    openGraph: {
      title: post.title,
      description: post.excerpt,
      type: "article",
    },
  };
};

const PostLayout = ({ params }: { params: { slug: string } }) => {
  const post = allPosts.find((post) => post._raw.flattenedPath === `blog/${params.slug}`);
  if (!post) {
    return notFound();
  }
  const Content = getMDXComponent(post.body.code);

  return (
    <article className="w-full max-w-4xl p-4 mx-auto prose lg:prose-lg">
      <div className="max-w-2xl py-8 mx-auto mb-8 ">
        <h1 className="text-center">{post.title}</h1>
        <span className="flex justify-center text-sm text-gray-600">
          <time dateTime={post.date} className="mx-2 ">
            Published on {new Date(post.date).toDateString()}
          </time>
          by {post.author}
        </span>
      </div>
      <Content />
    </article>
  );
};

export default PostLayout;
