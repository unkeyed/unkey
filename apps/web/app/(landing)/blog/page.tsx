import Link from "next/link";
import { allPosts, Post } from "contentlayer/generated";

export default function Blog() {
  const posts = allPosts.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime());

  return (
    <div className="py-24 sm:py-32">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl">
          <h2 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
            Unkey Blog
          </h2>

          <div className="mt-10 space-y-16 border-t border-gray-200 pt-10 sm:mt-16 sm:pt-16">
            {posts.map((post) => (
              <article
                key={post._id}
                className="flex max-w-xl flex-col items-start justify-between"
              >
                <div className="flex items-center gap-x-4 text-xs">
                  <time dateTime={post.date} className="text-gray-500">
                    {new Date(post.date).toDateString()}
                  </time>
                  {/* <a
                href={post.category.href}
                className="relative z-10 rounded-full bg-gray-50 px-3 py-1.5 font-medium text-gray-600 hover:bg-gray-100"
              >
                {post.category.title}
              </a> */}
                </div>
                <div className="group relative">
                  <h3 className="mt-3 text-lg font-semibold leading-6 text-gray-900 group-hover:text-gray-600">
                    <Link href={post.url}>
                      <span className="absolute inset-0" />
                      {post.title}
                    </Link>
                  </h3>
                  <p className="mt-5 line-clamp-3 text-sm leading-6 text-gray-600">
                    {post.excerpt}
                  </p>
                </div>
                {/* <div className="relative mt-8 flex items-center gap-x-4">
                       <img src={post.author.imageUrl} alt="" className="h-10 w-10 rounded-full bg-gray-50" />
                       <div className="text-sm leading-6">
                         <p className="font-semibold text-gray-900">
                           <a href={post.author.href}>
                             <span className="absolute inset-0" />
                             {post.author.name}
                           </a>
                         </p>
                         <p className="text-gray-600">{post.author.role}</p>
                       </div>
                     </div> */}
              </article>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
