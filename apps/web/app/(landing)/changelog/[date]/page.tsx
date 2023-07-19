//@ts-nocheck 
import { Container } from '@/components/landingComponents/Container'
import { FadeIn } from '@/components/landingComponents/FadeIn'
import { PageIntro } from '@/components/landingComponents/PageIntro'
import { PageLinks } from '@/components/landingComponents/PageLinks'
import { allChangelogs } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import { notFound } from "next/navigation";

export const generateStaticParams = async () =>
  allChangelogs.map((c) => ({
    date: new Date(c.date).toISOString().split("T")[0],
  }));


export default async function ChangeLogLayout({
  params,
}: {
  params: { date: string };
}) {

  const changelog = allChangelogs.find(
    (c) => new Date(c.date).toISOString().split("T")[0] === params.date,
  );

  if (!changelog || !changelog.body) {
    return notFound();
  }

  const Content = getMDXComponent(changelog.body.code);

  const moreChangelogs = allChangelogs.filter((p) => p.date !== params.date).slice(0, 2);

  return (
    <>
      <article className="mt-24 sm:mt-32 lg:mt-40">
        <header>
          <PageIntro eyebrow="" title={changelog.title} centered>
            <p>{changelog.description}</p>
          </PageIntro>

          <div className="mt-24 border-t border-neutral-200 bg-white/50 sm:mt-32 lg:mt-40">
            <Container>
              <div className="mx-auto max-w-5xl">
                <dl className="-mx-6 grid grid-cols-1 text-sm text-neutral-950 sm:mx-0 sm:grid-cols-3">
                  <div className="border-t border-neutral-200 px-6 py-4 first:border-t-0 sm:border-l sm:border-t-0">
                    <dt className="font-semibold">Number of changes</dt>
                    <dd>{changelog.changes}</dd>
                  </div>
                  <div className="border-t border-neutral-200 px-6 py-4 first:border-t-0 sm:border-l sm:border-t-0">
                    <dt className="font-semibold">Date</dt>
                    <dd>
                      <time dateTime={changelog.date}>
                        {changelog.date}
                      </time>
                    </dd>
                  </div>
                  <div className="border-t border-neutral-200 px-6 py-4 first:border-t-0 sm:border-l sm:border-t-0">
                    <dt className="font-semibold">New Features</dt>
                    <dd>{changelog.features}</dd>
                  </div>
                </dl>
              </div>
            </Container>
          </div>

          <div className="border-y border-neutral-200 bg-neutral-100">
            <div className="-my-px mx-auto max-w-[76rem] bg-neutral-200">
            </div>
          </div>
        </header>

        <Container className="mt-24 sm:mt-32 lg:mt-40">
          <FadeIn>
            <div className='mx-auto prose lg:prose-md max-w-5xl'>
              <Content />
            </div>
          </FadeIn>
        </Container>
      </article>

      {moreChangelogs.length > 0 && (
        <PageLinks
          className="mt-24 sm:mt-32 lg:mt-40"
          title="Read more changelogs"
          pages={moreChangelogs}
        />
      )}
    </>
  )
}
