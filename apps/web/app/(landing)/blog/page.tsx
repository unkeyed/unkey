import Link from "next/link";
import { compareDesc, format, parseISO } from "date-fns";
import { allPosts, Post } from "contentlayer/generated";

function PostCard(post: Post) {
  return (
    <Link
          href={post.url}>
    <div className="w-full flex flex-col max-w-xl">
      <h2 className="text-xl">
        
          {post.title}
        
      </h2>
      <time dateTime={post.date} className="block mb-2 text-xs text-gray-600">
        {format(parseISO(post.date), "LLLL d, yyyy")}
      </time>
      <div className="text-sm">
        <p>{post.excerpt}</p>
      </div>
    </div>
    </Link>
  );
}

export default function Home() {
  const posts = allPosts.sort((a, b) =>
    compareDesc(new Date(a.date), new Date(b.date))
  );

  return (
    <div className="max-w-3xl py-8 mx-auto">
      <h1 className="mb-8 text-3xl font-bold text-center">Unkey Blog </h1>

      {posts.map((post, idx) => (
        <PostCard key={idx} {...post} />
      ))}
    </div>
  );
}
