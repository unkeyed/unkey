// "use client";

// import { cn } from "@/lib/utils";

// import { format } from "date-fns";

// type BlogSuggestedProps = {
//   posts: Post[];
//   className?: string;
// };

// export function BlogCard({
//   posts,
//   className,
// }: BlogSuggestedProps) {

//   return (
//     <div className={cn("flex flex-col p-0 m-0 gap-8", className)}>
//       <p className="text-white/60">Suggested</p>
//       {posts.map((post) => (
//         <Link href={`/blog/${post.slug}`} key={post.slug}>
//           <div>
//             <Image
//               src={post.frontmatter.imageUrl}
//               width={1920}
//               height={1080}
//               alt="Hero Image"
//             />
//             <h3 className="font-medium text-3xl leading-10 blog-heading-gradient">
//               {post.frontmatter.title}
//             </h3>
//             <p className="text-white/40 text-sm pt-3 ml-6 font-normal">
//               {format(new Date(post.frontmatter.publishDate!), "MMM dd, yyyy")}
//             </p>
//           </div>
//         </Link>
//       ))}
//     </div>
//   );
// }
