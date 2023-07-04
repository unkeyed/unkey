import { allChangelogs, type Changelog } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import Link from "next/link";


export const metadata = {
  title: "Changelog | Unkey",
  description: "Changelog for Unkey",
  openGraph: {
    title: "Changelog | Unkey",
    description: "Changelog for Unkey",
    url: "https://unkey.dev/changelog",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/og?title=Changelog%20%7C%20Unkey",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/og.png",
  },
  robots: {
    index: true,
    follow: true,
    nocache: true,
    googleBot: {
      index: true,
      follow: false,
      noimageindex: true,
      'max-video-preview': -1,
      'max-image-preview': 'large',
      'max-snippet': -1,
    },
  },
};

export default function ChangelogPage() {
  const changelogs = allChangelogs.sort(
    (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
  );

  return (
    <div className="relative">
      <div className="px-4 sm:px-6 lg:px-8">
        <div className="relative mx-auto max-w-[37.5rem] pt-20 text-center pb-20">
          <h1 className="text-4xl font-extrabold tracking-tight text-slate-900 sm:text-5xl">
            Changelog
          </h1>
        </div>
      </div>
      <div className="relative max-w-5xl px-4 mx-auto sm:px-6 lg:px-8">
        {changelogs.map((c) => (
          <ChangelogSection key={c.date} changelog={c} />
        ))}
      </div>
    </div>
  );
}

type ChangelogSectionProps = {
  changelog: Changelog;
};
const ChangelogSection: React.FC<ChangelogSectionProps> = ({ changelog }) => {
  const Content = getMDXComponent(changelog.body.code);

  return (
    <section key={changelog.date} id={changelog.date} className="md:flex">
      <h2 className="text-sm leading-6 pl-7 text-slate-500 md:w-1/4 md:pl-0 md:pr-12 md:text-right">
        <Link href={`/changelog/${changelog.date}`}>{new Date(changelog.date).toDateString()}</Link>
      </h2>
      <div className="relative pt-2 pb-16 pl-7 md:w-3/4 md:pl-12 md:pt-0">
        <div className="absolute bottom-0 left-0 w-px bg-slate-200 -top-3 md:top-2.5" />
        <div className="absolute -left-1 -top-[1.0625rem] h-[0.5625rem] w-[0.5625rem] rounded-full border-2 border-slate-300 bg-white md:top-[0.4375rem]" />
        <div className="prose-sm prose max-w-none prose-h3:mb-4 prose-h3:text-base prose-h3:leading-6 prose-slate prose-a:font-semibold prose-a:text-sky-500 hover:prose-a:text-sky-600">
          <h2>{changelog.title}</h2>

          <Content />
        </div>
      </div>
    </section>
  );
};
