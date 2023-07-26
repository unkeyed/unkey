import Image from "next/image";
import Link from "next/link";

import { Border } from "@/components/landing/border";
import { Button } from "@/components/landing/button";
import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { PageIntro } from "@/components/landing/page-intro";
import { formatDate } from "@/lib/formatDate";
import { allPosts } from "contentlayer/generated";

export const metadata = {
  title: "Blog",
  description: "Stay up-to-date with the latest news and articles from the Unkey team.",
};

export default async function Blog() {
  const posts = allPosts.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
  return (
    <>
      <PageIntro eyebrow="Blog" title="The latest articles and news">
        <p>
          Stay up-to-date with the Unkey team as we share our latest news and articles in our
          industry.
        </p>
      </PageIntro>

      <Container className="mt-24 sm:mt-32 lg:mt-40">
        <div className="space-y-24 lg:space-y-32">
          {posts.map((post) => (
            <FadeIn key={post.url}>
              <article>
                <Border className={"pt-16"}>
                  <div className="relative lg:-mx-4 lg:flex lg:justify-end">
                    <div className="pt-10 lg:w-2/3 lg:flex-none lg:px-4 lg:pt-0">
                      <h2 className="text-2xl font-semibold font-display text-neutral-950">
                        <Link href={post.url}>{post.title}</Link>
                      </h2>
                      <dl className="lg:absolute lg:left-0 lg:top-0 lg:w-1/3 lg:px-4">
                        <dt className="sr-only">Published</dt>
                        <dd className="absolute top-0 left-0 text-sm text-neutral-950 lg:static">
                          <time dateTime={new Date(post.date).toDateString()}>
                            {new Date(post.date).toDateString()}
                          </time>
                        </dd>
                        <dt className="sr-only">Author</dt>
                        <dd className="flex mt-6 gap-x-4">
                          <div className="flex-none overflow-hidden rounded-xl bg-neutral-100">
                            <Image
                              alt={post.author.name}
                              {...post.author.image}
                              width={12}
                              height={12}
                              className="object-cover w-12 h-12 grayscale"
                            />
                          </div>
                          <div className="text-sm text-neutral-950">
                            <div className="font-semibold">{post.author.name}</div>
                            <div>{post.author.role}</div>
                          </div>
                        </dd>
                      </dl>
                      <p className="max-w-2xl mt-6 text-base text-neutral-600">
                        {post.description}
                      </p>
                      <Button
                        href={post.url}
                        aria-label={`Read more: ${post.title}`}
                        className="mt-8"
                      >
                        Read more
                      </Button>
                    </div>
                  </div>
                </Border>
              </article>
            </FadeIn>
          ))}
        </div>
      </Container>
    </>
  );
}
