import { ArrowRight, ChevronRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

import { BorderBeam } from "@/components/border-beam";
import { PrimaryButton } from "@/components/button";
import { RainbowDarkButton } from "@/components/button";
import { Container } from "@/components/container";
import { SectionTitle } from "@/components/section-title";
import { ChangelogLight } from "@/components/svg/changelog";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTriggerAbout,
} from "@/components/ui/accordion";
import { MeteorLines } from "@/components/ui/meteorLines";

import { BlogCard } from "@/app/blog/blog-card";
import { authors } from "@/content/blog/authors";
import andreas from "@/images/andreas.jpeg";
import carImage from "@/images/car-img.jpeg";
import james from "@/images/james.jpeg";
import { BLOG_PATH, getAllMDXData } from "@/lib/mdx-helper";

export default async function Page() {
  const posts = (await getAllMDXData({ contentPath: BLOG_PATH }))
    .sort((a, b) => {
      return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
    })
    .slice(0, 3);

  return (
    <Container>
      <div className="min-h-screen mt-[150px] flex flex-col items-center">
        <ChangelogLight />
        <div className="flex absolute -z-50">
          <div className="parallelogram">
            <BorderBeam size={300} delay={1} />
          </div>
          <div className="parallelogram parallelogram-1">
            <BorderBeam size={300} delay={0} />
          </div>
          <div className="parallelogram parallelogram-2">
            <BorderBeam size={300} delay={0.15} />
          </div>
          <div className="parallelogram parallelogram-3">
            <BorderBeam size={300} delay={0.3} />
          </div>
          <div className="parallelogram parallelogram-4">
            <BorderBeam size={300} delay={5} />
          </div>
        </div>
        <div className="mt-[120px]">
          <RainbowDarkButton label="We're hiring!" IconRight={ArrowRight} />
          <SectionTitle
            title="API auth for fast and scalable software"
            titleWidth={680}
            contentWidth={680}
            align="center"
            text="Unkey simplifies API authentication and authorization, making securing and managing APIs effortless. The platform delivers a fast and seamless developer experience for creating and verifying API keys, ensuring smooth integration and robust security."
          />
        </div>
        <div className="relative px-[144px] pb-[100px] pt-[60px] mt-[400px] text-white flex flex-col items-center rounded-[48px] border-l border-r border-b border-white/10 max-w-[1000px]">
          <div className="absolute left-[-250px]">
            <MeteorLines className="ml-2" delay={0.2} />
            <MeteorLines className="ml-10" />
            <MeteorLines className="ml-16" delay={0.4} />
          </div>
          <div className="absolute right-[20px]">
            <MeteorLines className="ml-2" delay={0.2} />
            <MeteorLines className="ml-10" />
            <MeteorLines className="ml-16" delay={0.4} />
          </div>

          {/* <div className="absolute right-[640px] top-[700px]">
          <MeteorLines className="ml-2" delay={0.2} />
          <MeteorLines className="ml-10" />
          <MeteorLines className="ml-16" delay={0.4} />
        </div> */}

          <h2 className="text-[32px] font-medium leading-[48px] mt-10 text-center">
            Founded to level up the API authentication landscape
          </h2>
          <p className="mt-[40px] text-white/50 leading-[32px] max-w-[720px] text-center">
            Unkey emerged in 2023 from the frustration of <span>James Perkins</span> and
            <span> Andreas Thomas</span> with the lack of a straightforward, fast, and scalable API
            authentication solution. This void prompted a mission to create a tool themselves. Thus,
            the platform was born, driven by their shared determination to simplify API
            authentication and democratize access for all developers. Today, the solution stands as
            a powerful tool, continuously evolving to meet the dynamic needs of a worldwide
            developer community.
          </p>
          <div className="absolute bottom-0">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="943"
              height="272"
              viewBox="0 0 943 272"
              fill="none"
            >
              <g style={{ mixBlendMode: "lighten" }} filter="url(#filter0_f_8003_5216)">
                <ellipse
                  cx="321.5"
                  cy="187.5"
                  rx="321.5"
                  ry="187.5"
                  transform="matrix(1 1.37458e-08 -1.37458e-08 -1 150 525.5)"
                  fill="url(#paint0_linear_8003_5216)"
                  fill-opacity="0.5"
                />
              </g>
              <defs>
                <filter
                  id="filter0_f_8003_5216"
                  x="0"
                  y="0.5"
                  width="943"
                  height="675"
                  filterUnits="userSpaceOnUse"
                  color-interpolation-filters="sRGB"
                >
                  <feFlood flood-opacity="0" result="BackgroundImageFix" />
                  <feBlend
                    mode="normal"
                    in="SourceGraphic"
                    in2="BackgroundImageFix"
                    result="shape"
                  />
                  <feGaussianBlur stdDeviation="75" result="effect1_foregroundBlur_8003_5216" />
                </filter>
                <linearGradient
                  id="paint0_linear_8003_5216"
                  x1="321.5"
                  y1="0"
                  x2="321.5"
                  y2="375"
                  gradientUnits="userSpaceOnUse"
                >
                  <stop stop-color="white" />
                  <stop offset="1" stop-color="white" stop-opacity="0" />
                </linearGradient>
              </defs>
            </svg>
          </div>
        </div>
        <SectionTitle
          className="mt-20"
          align="center"
          title="And now, we got people to take care of"
          titleWidth={640}
          contentWidth={640}
          text="We grew in number, and we love that. Here are some of our precious moments. Although we collaborate as a fully remote team, occasionally we unite!"
        />
        <div className="about-image-grid grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4 mt-[100px]">
          <div className="rounded-[20px] h-[480px] w-[262px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[320px] w-[262px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[480px] w-[262px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[320px] w-[262px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[480px] w-[262px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[480px] w-[262px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[440px] w-[262px] relative top-[-160px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[480px] w-[262px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[440px] w-[262px] relative top-[-160px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
          <div className="rounded-[20px] h-[480px] w-[262px]">
            <Image
              src={carImage}
              alt="Picture of a car"
              className="h-full object-cover rounded-[20px]"
            />
          </div>
        </div>
        <div>
          <SectionTitle
            title="Driven by values"
            className="mt-[200px]"
            contentWidth={640}
            align="center"
            text="Just as significant as the products we craft is the culture we cultivate - a culture defined by our unwavering commitment to our core values"
          />
          <div className="text-white mt-[88px] grid grid-cols-3 border-[1px] border-white/10 rounded-[24px] mb-10">
            <div className="flex flex-col justify-center items-center p-[40px] border-white/10 border-r-[1px] border-b-[0.75px] rounded-tl-[24px]">
              <div>
                <h3 className="text-[18px] font-medium">One-way hashed keys</h3>
                <p className="text-white/60 text-[15px]">
                  Enhance authentication security with one-way hashed keys, ensuring irrersible{" "}
                  encryption for sensitive information protection
                </p>
              </div>
            </div>
            <div className="flex flex-col justify-center items-center p-[40px] border-r-[1px] border-b-[0.75px] border-white/10">
              <div>
                <h3 className="text-[18px] font-medium">One-way hashed keys</h3>
                <p className="text-white/60 text-[15px]">
                  Enhance authentication security with one-way hashed keys, ensuring irrersible{" "}
                  encryption for sensitive information protection
                </p>
              </div>
            </div>
            <div className="flex flex-col justify-center items-center p-[40px] border-b-[0.75px] border-white/10 rounded-tr-[24px]">
              <div>
                <h3 className="text-[18px] font-medium">One-way hashed keys</h3>
                <p className="text-white/60 text-[15px]">
                  Enhance authentication security with one-way hashed keys, ensuring irrersible{" "}
                  encryption for sensitive information protection
                </p>
              </div>
            </div>
            <div className="flex flex-col justify-center items-center p-[40px] border-r-[1px] border-white/10 rounded-bl-[24px]">
              <div>
                <h3 className="text-[18px] font-medium">One-way hashed keys</h3>
                <p className="text-white/60 text-[15px]">
                  Enhance authentication security with one-way hashed keys, ensuring irrersible{" "}
                  encryption for sensitive information protection
                </p>
              </div>
            </div>
            <div className="flex flex-col justify-center items-center p-[40px] border-r-[1px] border-white/10">
              <div>
                <h3 className="text-[18px] font-medium">One-way hashed keys</h3>
                <p className="text-white/60 text-[15px]">
                  Enhance authentication security with one-way hashed keys, ensuring irrersible{" "}
                  encryption for sensitive information protection
                </p>
              </div>
            </div>
            <div className="flex flex-col justify-center items-center p-[40px] border-white/10 rounded-br-[24px]">
              <div>
                <h3 className="text-[18px] font-medium">One-way hashed keys</h3>
                <p className="text-white/60 text-[15px]">
                  Enhance authentication security with one-way hashed keys, ensuring irrersible{" "}
                  encryption for sensitive information protection
                </p>
              </div>
            </div>
          </div>
          <div className="flex flex-col items-center">
            <SectionTitle
              title="A few words from the founders"
              align="center"
              contentWidth={640}
              titleWidth={640}
              text="Take a peek into the minds behind Unkey. Here, our founders share their thoughts and stories, giving you a glimpse into what drives us forward."
            />
            <div className="border-t border-l border-r border-white/10 mt-[104px] rounded-[48px] py-[96px] px-[88px] text-white text-center max-w-[832px] flex flex-col justify-center items-center">
              <p>
                Nice to meet you! We're James and Andreas. We crossed paths while working together
                at a leading tech firm, where James led the design team, and Andreas was
                instrumental in developing cutting-edge design systems. It was there that the seeds
                were sown for what ultimately inspired us to launch Unkey. Below, we've compiled
                some questions we frequently encounter, as well as those we're eager to address.
              </p>
              <div className="flex mt-4">
                <div className="flex left-[5px]">
                  <div className="text-sm text-right">
                    <p className="font-bold">James Perkins</p>
                    <p className="text-white/40">Founder and CEO</p>
                  </div>
                  <Image
                    src={james}
                    className="border-2 border-black ml-[32px] rounded-full h-[40px] w-[40px]"
                    alt="CEO James"
                  />
                </div>
                <div className="flex relative left-[-5px]">
                  <Image
                    src={andreas}
                    alt="CTO Andreas"
                    className="border-2 border-black mr-[32px] rounded-full h-[40px] w-[40px]"
                  />
                  <div className="text-sm">
                    <p className="font-bold">Andreas Thomas</p>
                    <p className="text-white/40">Founder and CTO</p>
                  </div>
                </div>
              </div>
              <div className="relative ">
                <Accordion
                  type="single"
                  collapsible
                  className=" w-full z-50 mt-10 border border-white/10 rounded-[20px]"
                >
                  <AccordionItem value="item-1">
                    <AccordionTriggerAbout>What's your goal with Unkey?</AccordionTriggerAbout>
                    <AccordionContent>TBC</AccordionContent>
                  </AccordionItem>
                  <AccordionItem value="item-2">
                    <AccordionTriggerAbout>
                      What's something you're particularly happy about at Unkey?
                    </AccordionTriggerAbout>
                    <AccordionContent>TBC</AccordionContent>
                  </AccordionItem>
                  <AccordionItem value="item-3">
                    <AccordionTriggerAbout>
                      What's something you're less happy about?
                    </AccordionTriggerAbout>
                    <AccordionContent>TBC</AccordionContent>
                  </AccordionItem>
                </Accordion>
              </div>
            </div>
          </div>
          <div className="flex flex-col">
            <SectionTitle
              align="center"
              title="Backed by the finest"
              contentWidth={630}
              text="At Unkey, we're privileged to receive backing from top-tier investors, visionary founders, and seasoned operators from across the globe."
            />
            <div className="flex mx-auto">
              <div className="w-[320px] pt-[88px] px-[40px] pb-[80px]">
                <div className="text-[15px] text-center flex flex-col justify-center items-center">
                  <Image
                    src={james}
                    alt="Timothy Chen"
                    className="h-[48px] w-[48px] rounded-[100%]"
                  />
                  <p className="text-sm font-bold text-white pt-[32px]">Timothy Chen</p>
                  <p className="text-sm text-white/20">Essence VC</p>
                </div>
              </div>
              <div className="w-[320px] pt-[88px] px-[40px] pb-[80px]">
                <div className="text-[15px] text-center flex flex-col justify-center items-center">
                  <Image
                    src={james}
                    alt="Timothy Chen"
                    className="h-[48px] w-[48px] rounded-[100%]"
                  />
                  <p className="text-sm font-bold text-white pt-[32px]">Timothy Chen</p>
                  <p className="text-sm text-white/20">Essence VC</p>
                </div>
              </div>
              <div className="w-[320px] pt-[88px] px-[40px] pb-[80px]">
                <div className="text-[15px] text-center flex flex-col justify-center items-center">
                  <Image
                    src={james}
                    alt="Timothy Chen"
                    className="h-[48px] w-[48px] rounded-[100%]"
                  />
                  <p className="text-sm font-bold text-white pt-[32px]">Timothy Chen</p>
                  <p className="text-sm text-white/20">Essence VC</p>
                </div>
              </div>
            </div>
            <SectionTitle
              align="center"
              title="From our blog"
              text="Explore insights, tips, and updates directly from our team members"
            />
            <div className="flex w-full mx-auto gap-8">
              {posts.map((post) => {
                return (
                  <BlogCard
                    tags={post.frontmatter.tags?.toString()}
                    imageUrl={post.frontmatter.image ?? "/images/blog-images/defaultBlog.png"}
                    title={post.frontmatter.title}
                    subTitle={post.frontmatter.description}
                    author={authors[post.frontmatter.author]}
                    publishDate={post.frontmatter.date}
                  />
                );
              })}
            </div>
          </div>
          <SectionTitle
            align="center"
            className="mt-[200px]"
            title="Protect your API. Start today."
            titleWidth={507}
          >
            <div className="flex space-x-6 ">
              <Link key="get-started" href="/app">
                <PrimaryButton label="Start Now" IconRight={ChevronRight} />
              </Link>
            </div>
          </SectionTitle>
          <div className="mt-10 mb-[200px]">
            <p className="w-full mx-auto text-sm leading-6 text-center text-white/60">
              2500 verifications FREE per month.
            </p>
            <p className="w-full mx-auto text-sm leading-6 text-center text-white/60">
              No CC required.
            </p>
          </div>
        </div>
      </div>
    </Container>
  );
}
