import { allChangelogs, type Changelog } from "contentlayer/generated";
import { getMDXComponent } from "next-contentlayer/hooks";
import Link from "next/link";

export default function ChangelogPage() {
  const changelogs = allChangelogs.sort(
    (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
  );

  return (
    <div className="relative -mt-[5.75rem] overflow-hidden pt-[5.75rem]">
      <div className="px-4 sm:px-6 lg:px-8">
        <div className="relative mx-auto max-w-[37.5rem] pt-20 text-center pb-20">
          <h1 className="text-4xl font-extrabold tracking-tight text-slate-900 sm:text-5xl">
            Changelog
          </h1>
        </div>

      </div>
      <div className="relative mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
        {changelogs.map((c) => (
          <ChangelogSection changelog={c} />
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
      <h2
        className="pl-7 text-sm leading-6 text-slate-500 md:w-1/4 md:pl-0 md:pr-12 md:text-right"
      >
        <Link href={`/changelog/${changelog.date}`}>
          {new Date(changelog.date).toDateString()}
        </Link>
      </h2>
      <div className="relative pl-7 pt-2 md:w-3/4 md:pl-12 md:pt-0 pb-16">
        <div className="absolute bottom-0 left-0 w-px bg-slate-200 -top-3 md:top-2.5" />
        <div className="absolute -left-1 -top-[1.0625rem] h-[0.5625rem] w-[0.5625rem] rounded-full border-2 border-slate-300 bg-white md:top-[0.4375rem]" />
        <div className="max-w-none prose-h3:mb-4 prose-h3:text-base prose-h3:leading-6 prose-sm prose prose-slate prose-a:font-semibold prose-a:text-sky-500 hover:prose-a:text-sky-600">

          <h2>{changelog.title}</h2>

          <Content />
        </div>
      </div>
    </section>
  );
};
