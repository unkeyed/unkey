import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { ExternalLink, VenetianMask } from "lucide-react";
import Link from "next/link";

// We can convert this to something else later when we have too many.
const openSourceFriends = [
  {
    name: "Open Status",
    href: "https://openstatus.dev",
    image: "https://www.openstatus.dev/api/og",
    description:
      "OpenStatus is an open source alternative to your current monitoring service with a beautiful status page.",
  },
  {
    name: "Cal.com",
    href: "https://cal.com/",
    image: "https://cal.com/_next/image?url=%2Fapi%2Fsocial%2Fog%3Fpath%3D%2F&w=1200&q=100",
    description:
      "Open Source Scheduling: Send a link and meet or build an entire marketplace for humans to connect",
  },
  {
    name: "Documenso",
    href: "https://documenso.com/",
    image: "https://documenso.com/opengraph-image.jpg",
    description: "Documenso the Open Source DocuSign Alternative",
  },
  {
    name: "Hanko",
    href: "https://www.hanko.io/",
    image:
      "https://uploads-ssl.webflow.com/5e6f5bf4a2ae9702a833f3ee/6373564527b2911ddbb0d6e9_Thumbnail.png",
    description: "Hanko - Open source authentication beyond passwords",
  },
  {
    name: "GitWonk",
    href: "https://gitwonk.com/",
    image: "https://gitwonk.com/og.jpg",
    description:
      "GitWonk is an open-source, self-hosted alternative to GitBook, Confluence, and Archbee.",
  },
  {
    name: "Trigger.dev",
    href: "https://trigger.dev/",
    image: "https://trigger.dev/build/_assets/og-image-QGY42CUG.jpg",
    description:
      "GitWonk is an open-source, self-hosted alternative to GitBook, Confluence, and Archbee.",
  },
];

const OpenSourceFriends = async () => {
  return (
    <Container className="mt-24 sm:mt-32 lg:mt-40">
      <FadeIn>
        <div className="max-w-7xl mx-auto flex flex-col justify-center items-center">
          <h1 className="font-display text-5xl font-medium tracking-tight text-gray-950 [text-wrap:balance] sm:text-7xl font-sans">
            Our Open source friends
          </h1>
          <p className="font-sans text-gray-500 text-lg mt-2">
            Unkey finds inspiration in open-source projects. Here's a list of our favorite ones
          </p>
          <div className="grid w-full grid-cols-1 gap-8 lg:w-3/4 lauto-rows-fr lg:grid-cols-2 mt-12 md:mt-24">
            {openSourceFriends.map((friend) => (
              <Link
                key={friend.name}
                href={friend.href}
                className="flex flex-col items-start h-96 overflow-hidden duration-200 border border-gray-200 shadow rounded-xl hover:shadow-md hover:scale-[1.01]"
              >
                <div className="relative flex justify-center items-center h-full w-full aspect-[16/9] sm:aspect-[2/1] lg:aspect-[3/2]">
                  {friend.image ? (
                    <img
                      src={friend.image}
                      alt={friend.name}
                      className="object-cover w-full h-full"
                    />
                  ) : (
                    <VenetianMask className="w-16 h-16 text-gray-200" />
                  )}
                </div>
                <div className="flex flex-col justify-between h-full px-4 pb-4">
                  <div>
                    <h3 className="mt-3 text-lg font-semibold leading-6 text-gray-900 group-hover:text-gray-600 line-clamp-2">
                      {friend.name}
                    </h3>
                    <p className="mt-5 text-sm leading-6 text-gray-500 line-clamp-2">
                      {friend.description}
                    </p>
                  </div>
                  <div className="flex items-center justify-between mt-5">
                    <ExternalLink className="w-4 h-4 text-gray-400" />
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </div>
      </FadeIn>
    </Container>
  );
};

export default OpenSourceFriends;
