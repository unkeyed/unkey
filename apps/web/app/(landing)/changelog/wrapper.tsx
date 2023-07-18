//@ts-nocheck 
import { Container } from '@/components/landingComponents/Container'
import { FadeIn } from '@/components/landingComponents/FadeIn'
import { GrayscaleTransitionImage } from '@/components/landingComponents/GrayscaleTransitionImage'
import { MDXComponents } from '@/components/landingComponents/MDXComponents'
import { PageIntro } from '@/components/landingComponents/PageIntro'
import { PageLinks } from '@/components/landingComponents/PageLinks'
import { loadMDXMetadata } from '@/lib/loadMDXMetadata'

export default async function ChangeLogLayout({ children, _segments }) {
  let id = _segments.at(-2)
  let allChangelogs = await loadMDXMetadata('changelog')
  let changelog = allChangelogs.find((changelog) => changelog.id === id)
  let moreChangelogs = allChangelogs
    .filter((changelog) => changelog.id !== id)
    .slice(0, 2);
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
                <GrayscaleTransitionImage
                  {...changelog.image}
                  quality={90}
                  className="w-full"
                  sizes="(min-width: 1216px) 76rem, 100vw"
                  priority
                />
              </div>
            </div>
        </header>

        <Container className="mt-24 sm:mt-32 lg:mt-40">
          <FadeIn>
            <MDXComponents.wrapper>{children}</MDXComponents.wrapper>
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
