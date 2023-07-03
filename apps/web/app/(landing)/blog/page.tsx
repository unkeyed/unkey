import Link from "next/link";
import { allPosts } from "contentlayer/generated";

export default function Blog() {
  const posts = allPosts.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime());

  return (
    <div className="py-24 sm:py-32">
      <div className="px-6 mx-auto max-w-7xl lg:px-8">
        <div className="max-w-2xl mx-auto">
          <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
            Unkey Blog
          </h2>

          <div className="pt-10 mt-10 space-y-16 border-t border-gray-200 sm:mt-16 sm:pt-16">
            {posts.map((post) => (
              <article
                key={post._id}
                className="flex flex-col items-start justify-between max-w-xl"
              >
                <div className="flex items-center text-xs gap-x-4">
                  <time dateTime={post.date} className="text-gray-500">
                    {new Date(post.date).toDateString()}
                  </time>
                </div>
                <div className="relative group">
                  <h3 className="mt-3 text-lg font-semibold leading-6 text-gray-900 group-hover:text-gray-600">
                    <Link href={post.url}>
                      <span className="absolute inset-0" />
                      {post.title}
                    </Link>
                  </h3>
                  <p className="mt-5 text-sm leading-6 text-gray-600 line-clamp-3">
                    {post.excerpt}
                  </p>
                </div>
               
              </article>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
