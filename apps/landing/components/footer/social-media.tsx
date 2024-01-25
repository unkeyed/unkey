import clsx from "clsx";
import Link from "next/link";

export const socialMediaProfiles = [
  { title: "X(Twitter)", href: "https://twitter.com/unkeydev" },
  { title: "GitHub", href: "https://github.com/unkeyed" },
  { title: "OSS Friends", href: "/oss-friends" },
  {
    title: "Book a Call",
    href: "https://cal.com/team/unkey/unkey-chat??utm_source=banner&utm_campaign=oss",
  },
];

export function SocialMedia({
  className,
}: {
  className?: string;
  invert?: boolean;
}) {
  return (
    <ul role="list" className={clsx("flex gap-x-10", className)}>
      {socialMediaProfiles.map((socialMediaProfile) => (
        <li key={socialMediaProfile.title}>
          <Link
            href={socialMediaProfile.href}
            aria-label={socialMediaProfile.title}
            className={clsx()}
          />
        </li>
      ))}
    </ul>
  );
}
