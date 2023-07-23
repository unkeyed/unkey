import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { PageLinks } from "@/components/landing/page-links";
import { formatDate } from "@/lib/formatDate";
import { allPosts } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import Link from "next/link";
import { notFound } from "next/navigation";
export const generateStaticParams = async () =>
  allPosts.map((post) => ({ slug: post._raw.flattenedPath }));

const BlogArticleWrapper = ({ params }: { params: { slug: string } }) => {
  const post = allPosts.find((post) => post._raw.flattenedPath === `blog/${params.slug}`);
  if (!post) {
    return notFound();
  }

  const Content = getMDXComponent(post.body.code);

  // Find other articles to recommend at the bottom of the page that aren't the current article
  const moreArticles = allPosts
    .filter((p) => p._raw.flattenedPath !== post._raw.flattenedPath)
    .slice(0, 2);
  return (
    <>
      <Container as="article" className="mt-24 sm:mt-32 lg:mt-40 ">
        <FadeIn>
          <header className="flex flex-col max-w-5xl mx-auto text-center">
            <h1 className="font-sans mt-6 font-display text-5xl font-medium tracking-tight text-neutral-950 [text-wrap:balance] sm:text-6xl">
              {post.title}
            </h1>
            <time
              dateTime={new Date(post.date).toDateString()}
              className="order-first text-sm text-neutral-950"
            >
              {formatDate(new Date(post.date).toString())}
            </time>
            <p className="mt-6 text-sm font-semibold text-neutral-950">
              by {post.author.name}, {post.author.role}
            </p>
          </header>
        </FadeIn>

        <FadeIn>
          <div className="max-w-5xl mx-auto prose lg:prose-md">
            <Content />
          </div>
        </FadeIn>
      </Container>
      <FadeIn>
        <div className="py-24 mx-auto max-w-7xl sm:px-6 sm:py-32 lg:px-8">
          <div className="relative px-6 pt-16 overflow-hidden bg-gray-900 shadow-2xl isolate sm:rounded-xl sm:px-16 md:pt-24 lg:flex lg:gap-x-20 lg:px-24 lg:pt-0">
            <svg
              viewBox="0 0 1024 1024"
              className="absolute left-1/2 top-1/2 -z-10 h-[64rem] w-[64rem] -translate-y-1/2 [mask-image:radial-gradient(closest-side,white,transparent)] sm:left-full sm:-ml-80 lg:left-1/2 lg:ml-0 lg:-translate-x-1/2 lg:translate-y-0"
              aria-hidden="true"
            >
              <circle
                cx={512}
                cy={512}
                r={512}
                fill="url(#759c1415-0410-454c-8f7c-9a820de03641)"
                fillOpacity="0.7"
              />
              <defs>
                <radialGradient id="759c1415-0410-454c-8f7c-9a820de03641">
                  <stop stopColor="#6030B3" />
                  <stop offset={1} stopColor="#6030B3" />
                </radialGradient>
              </defs>
            </svg>
            <div className="max-w-md mx-auto text-center lg:mx-0 lg:flex-auto lg:py-32 lg:text-left">
              <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
                Accelerate your API development
              </h2>

              <div className="flex items-center justify-center mt-10 gap-x-6 lg:justify-start">
                <Link
                  href="/app"
                  className="rounded-md bg-white px-3.5 py-2.5 text-sm font-semibold text-gray-900 shadow-sm hover:bg-gray-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
                >
                  Get started
                </Link>
                <Link href="/docs" className="text-sm font-semibold leading-6 text-white">
                  Documentation <span aria-hidden="true">→</span>
                </Link>
              </div>
            </div>
            <div className="relative mt-16 h-80 lg:mt-8">
              <img
                className="absolute left-0 top-0 w-[57rem] max-w-none rounded-md bg-white/5 ring-1 ring-white/10"
                src="/images/landing/app.png"
                alt="App screenshot"
                width={1824}
                height={1080}
              />
            </div>
          </div>
        </div>
      </FadeIn>

      {moreArticles.length > 0 && (
        <PageLinks
          className="mt-24 sm:mt-32 lg:mt-40"
          title="More articles"
          intro=""
          pages={moreArticles}
        />
      )}
    </>
  );
};

export default BlogArticleWrapper;
