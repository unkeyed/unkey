import { ContactSection } from '@/components/landingComponents/ContactSection'
import { Container } from '@/components/landingComponents/Container'
import { FadeIn } from '@/components/landingComponents/FadeIn'
import { MDXComponents } from '@/components/landingComponents/MDXComponents'
import { PageLinks } from '@/components/landingComponents/PageLinks'
import { formatDate } from '@/lib/formatDate'
import { loadMDXMetadata } from '@/lib/loadMDXMetadata'

export default async function BlogArticleWrapper({ children, _segments } : { children: React.ReactNode, _segments: string[] }) {
  let id = _segments.at(-2)
  let allArticles = await loadMDXMetadata('blog')
  let article = allArticles.find((article) => article.id === id)
  let moreArticles = allArticles
    .filter((article) => article.id !== id)
    .slice(0, 2)
  console.log(id)
  return (
    <>
      <Container as="article" className="mt-24 sm:mt-32 lg:mt-40">
        <FadeIn>
          <header className="mx-auto flex max-w-5xl flex-col text-center">
            <h1 className="mt-6 font-display text-5xl font-medium tracking-tight text-neutral-950 [text-wrap:balance] sm:text-6xl">
              {article.title}
            </h1>
            <time
              dateTime={article.date}
              className="order-first text-sm text-neutral-950"
            >
              {formatDate(article.date)}
            </time>
            <p className="mt-6 text-sm font-semibold text-neutral-950">
              by {article.author.name}, {article.author.role}
            </p>
          </header>
        </FadeIn>

        <FadeIn>
          <MDXComponents.wrapper className="mt-24 sm:mt-32 lg:mt-40">
            {children}
          </MDXComponents.wrapper>
        </FadeIn>
      </Container>

      {moreArticles.length > 0 && (
        <PageLinks
          className="mt-24 sm:mt-32 lg:mt-40"
          title="More articles"
          intro=""
          pages={moreArticles}
        />
      )}
    </>
  )
}
