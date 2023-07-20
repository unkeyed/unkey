import Image from 'next/image'
import { Border } from '@/components/landing-components/border'
import { Container } from '@/components/landing-components/container'
import { FadeIn, FadeInStagger } from '@/components/landing-components/fade-in'
import { GridList, GridListItem } from '@/components/landing-components/grid-list'
import { PageIntro } from '@/components/landing-components/page-intro'
import { PageLinks } from '@/components/landing-components/page-links'
import { SectionIntro } from '@/components/landing-components/section-intro'
import { allPosts } from '@/.contentlayer/generated'
import imageJamesPerkins from '@/images/team/james.jpg'
import imageAndreas from '@/images/team/andreas.jpeg'
function AboutUnkey() {
  return (
    <div className="mt-24 rounded-4xl bg-neutral-950 py-24 sm:mt-32 lg:mt-40 lg:py-32 ">
      <SectionIntro
        eyebrow=""
        title="What is Unkey?"
        invert
      >
      </SectionIntro>
      <Container className="mt-16">
        <GridList>
          <GridListItem title="Globally Founded, Globally Remote" invert>
          We are globally remote and were founded that way too. We believe that the best talent is not always in the same place and that we can build a better product by hiring the best talent, no matter where they are.
          </GridListItem>
          <GridListItem title="Builders, Innovators" invert>
          We are serial builders who love to innovate. We are always looking for new ways to improve our product and our community. If we aren&apos;t working on Unkey, we are probably learning about something new.
          </GridListItem>
          <GridListItem title="Open Source" invert>
          Unkey is a fully open source project, we believe that open source leads to better products and better communities. We are committed to building a great open source community around Unkey and providing the ability to self host for those who want it.
          </GridListItem>
        </GridList>
      </Container>
    </div>
  )
}

const team = [
  {
    title: 'Team',
    people: [
      {
        name: 'James Perkins',
        role: 'Co-Founder / CEO',
        image: { src: imageJamesPerkins },
      },
      {
        name: 'Andreas Thomas',
        role: 'Co-Founder / CTO',
        image: { src: imageAndreas },
      },
    ],
  },
]

function Team() {
  return (
    <Container className="mt-24 sm:mt-32 lg:mt-40">
      <div className="space-y-24">
        {team.map((group) => (
          <FadeInStagger key={group.title}>
            <Border as={FadeIn} />
            <div className="grid grid-cols-1 gap-6 pt-12 sm:pt-16 lg:grid-cols-4 xl:gap-8">
              <FadeIn>
                <h2 className="font-display text-2xl font-semibold text-neutral-950 font-sans">
                  {group.title}
                </h2>
              </FadeIn>
              <div className="lg:col-span-3">
                <ul
                  role="list"
                  className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:gap-8"
                >
                  {group.people.map((person) => (
                    <li key={person.name}>
                      <FadeIn>
                        <div className="group relative overflow-hidden rounded-3xl bg-neutral-100">
                          <Image
                            alt=""
                            {...person.image}
                            className="h-96 w-full object-cover grayscale transition duration-500 motion-safe:group-hover:scale-105"
                          />
                          <div className="absolute inset-0 flex flex-col justify-end bg-gradient-to-t from-black to-black/0 to-40% p-6">
                            <p className="font-display text-base/6 font-semibold tracking-wide text-white">
                              {person.name}
                            </p>
                            <p className="mt-2 text-sm text-white">
                              {person.role}
                            </p>
                          </div>
                        </div>
                      </FadeIn>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </FadeInStagger>
        ))}
      </div>
    </Container>
  )
}

export const metadata = {
  title: 'About Us',
  description:
    'We believe that our strength lies in our collaborative approach, which puts our clients at the center of everything we do.',
}

export default async function About() {
  const blogArticles = allPosts.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime()).slice(0,2);
  return (
    <>
      <PageIntro eyebrow="" title="About us">
        <p>
          Unkey is a fully open source project, we believe that open source leads to better products and better communities. We are committed to building a great open source community around Unkey and providing the ability to self host for those who want it.
        </p>
        <div className="mt-10 max-w-2xl space-y-6 text-base">
          <p>
            Unkey was started by James Perkins and Andreas Thomas in 2023. We are a small team of serial builders who love to innovate. We are always looking for new ways to improve our product and our community. If we aren&apos;t working on Unkey, we are probably learning about something new.
          </p>
        </div>
      </PageIntro>


      <AboutUnkey />

      <Team />

      <PageLinks
        className="mt-24 sm:mt-32 lg:mt-40"
        title="From the blog"
        intro="We write about latest trends in development, and changes to Unkey"
        pages={blogArticles}
      />
    </>
  )
}
