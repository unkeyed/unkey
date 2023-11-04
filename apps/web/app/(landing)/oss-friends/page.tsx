import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";
import { ExternalLink } from "lucide-react";
import Link from "next/link";

type OSSFriend = {
  href: string;
  name: string;
  description: string;
};

// We can convert this to something else later when we have too many.
const res = await fetch("https://formbricks.com/api/oss-friends");
const data = await res.json();
const OSSFriends = data.data;

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
            {OSSFriends.map((friend: OSSFriend) => (
              <Link
                key={friend.name}
                href={friend.href}
                className="flex flex-col items-start overflow-hidden duration-200 border border-gray-200 shadow rounded-xl hover:shadow-md hover:scale-[1.01]"
              >
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
