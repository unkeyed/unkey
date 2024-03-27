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
import { AboutLight } from "@/components/svg/about-light";
import { authors } from "@/content/blog/authors";
import allison from "@/images/about/allison5.png";
import bottomlight from "@/images/about/bottomlight.svg";
import downlight from "@/images/about/down-light.svg";
import placeholder from "@/images/about/landscape-placeholder.svg";
import liu from "@/images/about/liujiang.jpeg";
import sidelight from "@/images/about/side-light.svg";
import tim from "@/images/about/tim.png";
import andreas from "@/images/team/andreas.jpeg";
import james from "@/images/team/james.jpg";
import { BLOG_PATH, getAllMDXData } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";

const investors = [
  { name: "Timothy Chen", firm: "Essence VC", image: tim },
  { name: "Liu Jiang", firm: "Sunflower Capital", image: liu },
  { name: "Allison Pickets", firm: "The New Normal Fund", image: allison },
];

const SELECTED_POSTS = ["uuid-ux", "why-we-built-unkey", "unkey-raises-1-5-million"];

export default async function Page() {
  const selectedPosts = (await getAllMDXData({ contentPath: BLOG_PATH }))
    .sort((a, b) => {
      return new Date(b.frontmatter.date).getTime() - new Date(a.frontmatter.date).getTime();
    })
    .filter((post) => SELECTED_POSTS.includes(post.slug));

  return (
    <Container>
      <div className="mt-[150px] flex flex-col items-center">
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
          <Link href="/careers" target="">
            <RainbowDarkButton label="We are hiring!" IconRight={ArrowRight} />
          </Link>
          <SectionTitle
            title="API auth for fast and scalable software"
            titleWidth={680}
            contentWidth={680}
            align="center"
            text="Unkey simplifies API authentication and authorization, making securing and managing APIs effortless. The platform delivers a fast and seamless developer experience for creating and verifying API keys, ensuring smooth integration and robust security."
          />
        </div>
        <div className="relative mt-[200px] xl:mt-[400px]">
          <div className="absolute left-[-250px]">
            <MeteorLines className="ml-2" delay={0.2} />
            <MeteorLines className="ml-10" />
            <MeteorLines className="ml-16" delay={0.4} />
          </div>
          <div className="absolute right-[20px]">
            <MeteorLines className="ml-2" delay={0.2} />
            <MeteorLines className="ml-10" />
            <MeteorLines className="ml-16" delay={0.4} />

            {/* <div className="absolute right-[640px] top-[700px]">
              <MeteorLines className="ml-2" delay={0.2} />
              <MeteorLines className="ml-10" />
              <MeteorLines className="ml-16" delay={0.4} />
            </div> */}
          </div>
          <div className="relative px-[50px] md:px-[144px] pb-[100px] pt-[60px] overflow-hidden text-white flex flex-col items-center rounded-[48px] border-l border-r border-b border-white/20 max-w-[1000px]">
            <h2 className="text-[32px] font-medium leading-[48px] mt-10 text-center">
              Founded to level up the API authentication landscape
            </h2>
            <p className="mt-[40px] text-white/50 leading-[32px] max-w-[720px] text-center">
              Unkey emerged in 2023 from the frustration of <span>James Perkins</span> and
              <span> Andreas Thomas</span> with the lack of a straightforward, fast, and scalable
              API authentication solution. This void prompted a mission to create a tool themselves.
              Thus, the platform was born, driven by their shared determination to simplify API
              authentication and democratize access for all developers. Today, the solution stands
              as a powerful tool, continuously evolving to meet the dynamic needs of a worldwide
              developer community
            </p>
            <div className="absolute scale-[1.5] bottom-[-350px]">
              <AboutLight />
            </div>
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
        <div className="grid about-image-grid grid-cols-1 md:grid-cols-3 xl:grid-cols-5 gap-4 mt-[62px]">
          <div className="image w-[200px] h-[400px] rounded-lg relative">
            <PhotoLabel
              className="absolute bottom-[40px] left-[calc(50%-40px)]"
              text="Label text"
            />
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
          <div className="image w-[200px] h-[400px] hidden md:block xl:hidden bg-black rounded-lg" />
          <div className="image w-[200px] h-[400px] rounded-lg">
            <Image
              src={placeholder}
              alt="photo of a car"
              className="h-full w-full object-cover rounded-lg"
            />
          </div>
        </div>

        <div className="relative w-screen max-w-full">
          <Image src={sidelight} alt="lightbeam effect" className="absolute right-[-300px]" />
          <Image
            src={sidelight}
            alt="lightbeam effect"
            className="absolute right-0 scale-x-[-1] left-[-300px]"
          />
          <SectionTitle
            title="Driven by values"
            className="mt-[200px] max-w-full"
            contentWidth={640}
            align="center"
            text="Just as significant as the products we craft is the culture we cultivate - a culture defined by our unwavering commitment to our core values"
          />
          <div className="mx-auto px-6 lg:px-8">
            <div className="text-white mt-[62px] w-full grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 border-[1px] border-white/10 rounded-[24px] mb-10">
              {Array.from({ length: 6 }).map(() => {
                return (
                  <div className="flex flex-col justify-center items-center p-[40px] border-white/10 border-r-[1px] border-b-[0.75px] rounded-tl-[24px]">
                    <div>
                      <h3 className="text-[18px] font-medium">One-way hashed keys</h3>
                      <p className="text-white/60 text-[15px] leading-6 lg:max-w-[4500px] xl:max-w-[280px] pt-2">
                        Enhance authentication security with one-way hashed keys, ensuring
                        irrersible encryption for sensitive information protection
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
        <div className="flex flex-col items-center max-w-full">
          <SectionTitle
            className="mt-[100px] px-[10px]"
            title="A few words from the founders"
            align="center"
            contentWidth={640}
            titleWidth={640}
            text="Take a peek into the minds behind Unkey. Here, our founders share their thoughts and stories, giving you a glimpse into what drives us forward."
          />
          <div className="border-[1px] border-white/10 mt-[78px] leading-8 rounded-[48px] py-[60px] xl:py-[96px] px-8 md:px-[88px] text-white text-center max-w-[1008px] flex flex-col justify-center items-center">
            <p className="about-founders-text-gradient">
              Nice to meet you! We're James and Andreas. We crossed paths while working together at
              a leading tech firm, where James led the design team, and Andreas was instrumental in
              developing cutting-edge design systems. It was there that the seeds were sown for what
              ultimately inspired us to launch Unkey. Below, we've compiled some questions we
              frequently encounter, as well as those we're eager to address.
            </p>
            <div className="flex flex-col md:flex-row mt-8">
              <div className="flex md:left-[5px]">
                <div className="text-sm text-right">
                  <p className="font-bold">James Perkins</p>
                  <p className="text-white/40">Founder and CEO</p>
                </div>
                <Image
                  src={james}
                  className="border-2 border-black ml-4 md:ml-[32px] rounded-full h-[40px] w-[40px]"
                  alt="CEO James"
                />
              </div>
              <div className="flex relative mt-4 md:mt-0 lg:left-[-5px]">
                <Image
                  src={andreas}
                  alt="CTO Andreas"
                  className="border-2 border-black mr-4 md:mr-[32px] rounded-full h-[40px] w-[40px]"
                />
                <div className="text-sm">
                  <p className="font-bold">Andreas Thomas</p>
                  <p className="text-white/40">Founder and CTO</p>
                </div>
              </div>
            </div>
          </div>
          <div className="relative w-full max-w-[680px] z-0">
            <div className="relative z-50 bg-black w-full">
              <Accordion
                type="single"
                collapsible
                className="relative w-full z-50 mt-12 border border-white/10 rounded-[20px] text-white"
              >
                <AccordionItem
                  value="item-1"
                  className="border border-white/10 rounded-tr-[20px] rounded-tl-[20px]"
                >
                  <AccordionTriggerAbout>What's your goal with Unkey?</AccordionTriggerAbout>
                  <AccordionContent className="pl-10">TBC</AccordionContent>
                </AccordionItem>
                <AccordionItem value="item-2" className="border border-white/10">
                  <AccordionTriggerAbout>
                    What's something you're particularly happy about at Unkey?
                  </AccordionTriggerAbout>
                  <AccordionContent className="pl-10">TBC</AccordionContent>
                </AccordionItem>
                <AccordionItem
                  value="item-3"
                  className="border border-white/10 rounded-br-[20px] rounded-bl-[20px]"
                >
                  <AccordionTriggerAbout>
                    What's something you're less happy about?
                  </AccordionTriggerAbout>
                  <AccordionContent className="pl-10">TBC</AccordionContent>
                </AccordionItem>
              </Accordion>
            </div>
            <div className="absolute -z-50 hidden lg:flex lg:bottom-[-360px] lg:left-[100px]">
              <Image src={downlight} alt="Light effect" className="scale-[1.5]" />
            </div>
          </div>

          <div className="flex flex-col max-w-full">
            <SectionTitle
              className="mt-[250px]"
              align="center"
              title="Backed by the finest"
              contentWidth={630}
              text="At Unkey, we're privileged to receive backing from top-tier investors, visionary founders, and seasoned operators from across the globe."
            />
            <div className="flex flex-col lg:flex-row lg:gap-x-16 mx-auto">
              {investors.map(({ name, firm, image }) => {
                return (
                  <div className="pt-[88px] px-[40px] pb-[80px]">
                    <div className="text-[15px] text-center flex flex-col justify-center items-center">
                      <Image
                        src={image}
                        alt="Liu Jiang"
                        className="h-[48px] w-[48px] rounded-[100%]"
                      />
                      <p className="text-sm font-bold text-white pt-[32px]">{name}</p>
                      <p className="text-sm text-white/20">{firm}</p>
                    </div>
                  </div>
                );
              })}
            </div>
            <div className="w-full h-[0.75px] bg-gradient-to-r from-white/10 to-white/10 via-white/40 mt-[100px] lg:mt-[200px]" />
            <SectionTitle
              className="mt-[100px] lg:mt-[200px]"
              align="center"
              title="From our blog"
              text="Explore insights, tips, and updates directly from our team members"
            />
            <div className="flex flex-col lg:flex-row w-full mx-auto gap-8 mt-[96px]">
              {selectedPosts.map((post) => {
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
      <Image
        src={bottomlight}
        alt="light effect"
        className="absolute bottom-[-90px] scale-[1.5] left-[0px] lg:bottom-[-280px] lg:left-[270px]"
      />
    </Container>
  );
}

function PhotoLabel({ text, className }: { text: string; className: string }) {
  return (
    <div
      className={cn(
        className,
        "bg-gradient-to-r from-black/70 to-black/40 px-4 py-1.5 rounded-[6px] backdrop-blur-md border-[0.75px] border-white/20",
      )}
    >
      <p className="text-white text-[13px]">{text}</p>
    </div>
  );
}
