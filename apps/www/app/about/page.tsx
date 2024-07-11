import { ArrowRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

import { BorderBeam } from "@/components/border-beam";
import { RainbowDarkButton } from "@/components/button";
import { Container } from "@/components/container";
import { SectionTitle } from "@/components/section";
import { ChangelogLight } from "@/components/svg/changelog";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTriggerAbout,
} from "@/components/ui/accordion";
import { MeteorLines } from "@/components/ui/meteorLines";

import { BlogCard } from "@/components/blog/blog-card";
import { CTA } from "@/components/cta";
import { AboutLight } from "@/components/svg/about-light";
import { StarDots } from "@/components/svg/star-dots";
import { authors } from "@/content/blog/authors";

import { allPosts } from "@/.contentlayer/generated";

import downlight from "@/images/about/down-light.svg";
import sidelight from "@/images/about/side-light.svg";

import andrew from "@/images/about/angels/andrew.jpeg";
import ant from "@/images/about/angels/ant.jpeg";
import paul from "@/images/about/angels/copple.jpeg";
import rory from "@/images/about/angels/rory.png";
import theo from "@/images/about/angels/theo.jpeg";
import tom from "@/images/about/angels/tom.jpeg";
import zain from "@/images/about/angels/zain.jpeg";
import allison from "@/images/about/investors/allison5.png";
import liu from "@/images/about/investors/liujiang.jpeg";
import tim from "@/images/about/investors/tim.png";

import art_intensifies from "@/images/offsite/art_intensifies.jpg";
import breakfast from "@/images/offsite/breakfast.jpg";
import cooking_crew from "@/images/offsite/cooking_crew.jpg";
import cto_prayers_answered from "@/images/offsite/cto_prayers_answered.jpg";
import dom_thinking from "@/images/offsite/dom_thinking.jpg";
import doomsday from "@/images/offsite/doomsday.jpg";
import james_fence from "@/images/offsite/james_fence.jpg";
import james_thinking from "@/images/offsite/james_thinking.jpg";
import mike_morning_neck_exercise from "@/images/offsite/mike_morning_neck_exercise.jpg";
import yardwork from "@/images/offsite/yardwork.jpg";
import andreas from "@/images/team/andreas.jpeg";
import james from "@/images/team/james.jpg";

import { ImageWithBlur } from "@/components/image-with-blur";
import { cn } from "@/lib/utils";

export const metadata = {
  title: "About | Unkey",
  description: "Learn more about Unkey and how we operate.",
  openGraph: {
    title: "About | Unkey",
    description: "Learn more about Unkey and how we operate.",
    url: "https://unkey.com/templates",
    siteName: "unkey.com",
    images: [
      {
        url: "https://unkey.com/images/landing/og.png",
        width: 1200,
        height: 675,
      },
    ],
  },
  twitter: {
    title: "About | Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/images/landing/unkey.png",
  },
};

const investors = [
  { name: "Timothy Chen", firm: "Essence VC", image: tim },
  { name: "Liu Jiang", firm: "Sunflower Capital", image: liu },
  { name: "Allison Pickens", firm: "The New Normal Fund", image: allison },
  { name: "Andrew Miklas", firm: "Ex PageDuty CTO", image: andrew },
  { name: "Tom Preston-Werner", firm: "GitHub founder", image: tom },
  { name: "Theo Browne", firm: "CEO @ Ping Labs", image: theo },
  { name: "Paul Copplestone", firm: "CEO @ Supabase", image: paul },
  { name: "Ant Wilson", firm: "CTO @ Supabase", image: ant },
  { name: "Rory Wilding", firm: "Head of Growth @ Supabase", image: rory },
  { name: "Zain Allarakhia", firm: "Former CTO @ Pipe", image: zain },
];

const SELECTED_POSTS = ["uuid-ux", "why-we-built-unkey", "unkey-raises-1-5-million"];

const offsiteImages = [
  { src: breakfast, label: "Cooking breakfast" },
  { src: art_intensifies, label: "Art intensifies" },
  { src: dom_thinking, label: "Hard at work" },
  { src: cooking_crew, label: "Lunch refuel" },
  { src: cto_prayers_answered, label: "Golden hour" },
  { src: doomsday, label: "Escape room W" },
  { src: james_fence, label: "James recruiting" },
  { src: james_thinking, label: "Deep in thought", className: "object-left" },
  {
    src: mike_morning_neck_exercise,
    label: "CEO + CTO",
    className: "object-left",
  },
  { src: yardwork, label: "Caffeinated" },
];

export default async function Page() {
  const posts = allPosts.filter((post) =>
    SELECTED_POSTS.includes(post._raw.flattenedPath.replace("blog/", "")),
  );
  return (
    <div>
      <Container>
        <div className="mt-[215px] flex flex-col items-center mb-[200px]">
          <ChangelogLight />
          <div className="absolute flex -z-50">
            <div className="parallelogram">{/* <BorderBeam size={300} delay={1} /> */}</div>
            <div className="parallelogram parallelogram-1">
              <BorderBeam size={300} anchor={150} />
            </div>
            <div className="parallelogram parallelogram-2">
              <BorderBeam size={300} anchor={100} />
            </div>
            <div className="parallelogram parallelogram-3">
              <BorderBeam size={300} anchor={50} />
            </div>
            <div className="parallelogram parallelogram-4">
              <BorderBeam size={300} anchor={0} />
            </div>
          </div>
          <div className="mt-12">
            <Link href="/blog/introducing-ratelimiting" target="">
              <RainbowDarkButton label="New: global rate limiting" IconRight={ArrowRight} />
            </Link>
            <SectionTitle
              title="API management for fast and scalable software"
              align="center"
              text="Unkey redefines API management for developers. You can add authentication, analytics, and rate-limiting to your APIs in minutes. "
            />
          </div>
          <div className="relative mt-[200px] xl:mt-[400px]">
            <div className="absolute left-[-250px]">
              <MeteorLines className="ml-2 fade-in-0" delay={3} number={1} />
              <MeteorLines className="ml-10 fade-in-40" delay={0} number={1} />
              <MeteorLines className="ml-16 fade-in-100" delay={5} number={1} />
            </div>
            <div className="absolute right-[20px]">
              <MeteorLines className="ml-2 fade-in-0" delay={4} number={1} />
              <MeteorLines className="ml-10 fade-in-40" delay={0} number={1} />
              <MeteorLines className="ml-16 fade-in-100" delay={2} number={1} />

              <div className="absolute right-[640px] top-[550px] -z-50">
                <MeteorLines className="ml-2 fade-in-0" delay={2} number={1} />
                <MeteorLines className="ml-10 fade-in-40" number={1} delay={0} />
                <MeteorLines className="ml-16 fade-in-100" delay={4} number={1} />
              </div>
            </div>
            <div className="about-radial relative px-[50px] md:px-[144px] pb-[100px] pt-[60px] overflow-hidden bg-black text-white flex flex-col items-center rounded-[48px] border-l border-r border-b border-white/[0.15]">
              <h2 className="text-[32px] font-medium leading-[48px] mt-10 text-center text-balance">
                Founded to redefine the API management landscape
              </h2>
              <p className="mt-[40px] text-white/50 leading-[32px] max-w-[720px] text-center">
                Unkey emerged in 2023 from the frustration of{" "}
                <span className="font-medium text-white">James Perkins</span> and
                <span className="font-medium text-white"> Andreas Thomas</span> with the lack of a
                straightforward, fast, and scalable API management solution. This void prompted a
                mission to create a tool themselves.
              </p>
              <div className="absolute pointer-events-none scale-[1.5] bottom-[-350px]">
                <AboutLight />
              </div>
            </div>
          </div>
          <SectionTitle
            className="mt-80"
            align="center"
            title="Meet the team"
            text="Although we collaborate as a fully remote team, we like to unite for regular offsites. Here are a few moments from our most recent:"
          />
          <div className="grid about-image-grid grid-cols-1 md:grid-cols-3 xl:grid-cols-5 gap-4 mt-[62px] w-full xl:w-[calc(100dvw-10rem)]">
            {offsiteImages.map(({ src, label, className }) => {
              return (
                <div className="image w-full h-[400px] rounded-lg relative">
                  <PhotoLabel
                    className="absolute bottom-[40px] left-1/2 transform -translate-x-1/2"
                    text={label}
                  />
                  <ImageWithBlur
                    src={src}
                    alt={label}
                    className={cn("object-cover w-full h-full rounded-lg", className)}
                  />
                </div>
              );
            })}
          </div>

          <div className="relative w-screen max-w-full">
            <Image
              src={sidelight}
              alt="lightbeam effect"
              className="absolute right-[-300px] pointer-events-none"
            />
            <Image
              src={sidelight}
              alt="lightbeam effect"
              className="absolute right-0 scale-x-[-1] left-[-300px] pointer-events-none"
            />
            <SectionTitle
              title="Driven by values"
              className="mt-[200px] max-w-full"
              align="center"
              text="Just as significant as the products we craft is the culture we cultivate - a culture defined by our unwavering commitment to our core values"
            />
            <div className="mx-auto md:px-5 lg:px-8">
              <div className="bg-white/10 overflow-hidden text-white mt-[62px] w-full gap-px grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 border-[1px] border-transparent rounded-3xl mb-10 ">
                <Value
                  text="We don't meet expectations; we redefine them by doing all the hard work upfront to craft an effortless user experience."
                  title="Quality"
                />
                <Value
                  text="Our default is to be open rather than closed. Simplicity, transparency, and honesty lead to the best results."
                  title="Open company"
                />
                <Value
                  title="Ownership"
                  text="Our team members are given a high degree of autonomy to develop, implement, and iterate on their ideas."
                />
                <Value
                  text="We prioritize quality while ensuring our team has a work-life balance that ensures they can deliver maximum value."
                  title="Sustainability"
                />
                <Value
                  text="We ship fast and work together with our users to solve real problems."
                  title="Customer obsessed"
                />
                <Value
                  text="We take security seriously and don't compromise in favour of velocity or user experience."
                  title="Security first"
                />
              </div>
            </div>
          </div>
          <div className="flex flex-col items-center max-w-full">
            <StarDots className="absolute pointer-events-none" />
            <SectionTitle
              className="mt-60 px-[10px] text-balance"
              title="A few words from the founders"
              align="center"
              text="Why we started Unkey and what we believe in."
            />
            <div className="border-[1px] border-white/10 mt-[78px] leading-8 rounded-[48px] py-[60px] xl:py-[96px] px-8 md:px-[88px] text-white text-center max-w-[1008px] flex flex-col justify-center items-center">
              <p className="about-founders-text-gradient">
                We're James and Andreas. We founded Unkey with the vision of creating an API
                management platform that is both powerful and easy to use. We believe that APIs are
                the building blocks of the modern web, and we want to make it easier for developers
                to build and manage them.
              </p>
              <div className="flex flex-col mt-12 md:flex-row">
                <div className="flex md:left-[5px]">
                  <div className="text-sm text-right">
                    <p className="font-bold">James Perkins</p>
                    <p className="text-white/50">Founder and CEO</p>
                  </div>
                  <ImageWithBlur
                    src={james}
                    className="border-2 border-black ml-4 md:ml-[32px] rounded-full h-[40px] w-[40px]"
                    alt="CEO James"
                  />
                </div>
                <div className="flex relative mt-4 md:mt-0 lg:left-[-5px]">
                  <ImageWithBlur
                    src={andreas}
                    alt="CTO Andreas"
                    className="border-2 border-black mr-4 md:mr-[32px] rounded-full h-[40px] w-[40px]"
                  />
                  <div className="text-sm text-left">
                    <p className="font-bold">Andreas Thomas</p>
                    <p className="text-white/50">Founder and CTO</p>
                  </div>
                </div>
              </div>
            </div>
            <div className="relative w-full max-w-[680px] z-0">
              <div className="relative w-full bg-black z-100">
                <Accordion
                  type="multiple"
                  className="relative w-full z-50 mt-12 border border-white/10 rounded-[20px] text-white"
                >
                  <AccordionItem
                    value="item-1"
                    className="border border-white/10 rounded-tr-[20px] rounded-tl-[20px]"
                  >
                    <AccordionTriggerAbout>What's your goal with Unkey?</AccordionTriggerAbout>
                    <AccordionContent className="pl-10">
                      Our goal with Unkey is build an open source API management platform that
                      doesn’t require the burden or cost of traditional API management platforms
                      like Kong or Azure APM. We want to embrace what developers know today, a
                      global REST API that allows you to deploy and protect your API on the edge in
                      under 5 minutes.{" "}
                    </AccordionContent>
                  </AccordionItem>
                  <AccordionItem value="item-2" className="border border-white/10">
                    <AccordionTriggerAbout>
                      What's something you're particularly happy about at Unkey?
                    </AccordionTriggerAbout>
                    <AccordionContent className="pl-10">
                      We are extremely happy with the culture we have built at Unkey, our team is
                      small but powerful. Everyone in our team has input on the next feature or idea
                      we have for Unkey, allowing us to build the best API management platform.{" "}
                    </AccordionContent>
                  </AccordionItem>
                  <AccordionItem
                    value="item-3"
                    className="border border-white/10 rounded-br-[20px] rounded-bl-[20px]"
                  >
                    <AccordionTriggerAbout>
                      What's something you're less happy about?
                    </AccordionTriggerAbout>
                    <AccordionContent className="pl-10">
                      While we are happy with Unkey overall knowing when to build tall or wide is a
                      problem we run into frequently. Having a small team and focusing on incredible
                      DX means that adding new features or improving currents ones means that we
                      have to be certain that it will bring value to our users.{" "}
                    </AccordionContent>
                  </AccordionItem>
                </Accordion>
                <div className="absolute right-[500px] top-[150px] -z-50">
                  <MeteorLines className="ml-2 fade-in-0" delay={3} number={1} />
                  <MeteorLines className="ml-10 fade-in-40" delay={0} number={1} />
                  <MeteorLines className="ml-16 fade-in-100" delay={5} number={1} />
                </div>
              </div>
              <div className="absolute pointer-events-none -z-50 hidden lg:flex lg:bottom-[-360px] lg:left-[100px]">
                <Image src={downlight} alt="Light effect" className="scale-[1.5] opacity-[0.7]" />
              </div>
            </div>

            <div className="flex flex-col max-w-full">
              <SectionTitle
                className="mt-[250px]"
                align="center"
                title="Backed by the finest"
                text="At Unkey, we're privileged to receive backing from top-tier investors, visionary founders, and seasoned operators from across the globe. Here are just a few of them: "
              />
              <div className="grid justify-center w-full grid-cols-2 pt-24 mx-auto md:grid-cols-3 lg:grid-cols-5 ">
                {investors.map(({ name, firm, image }) => {
                  return (
                    <div className="flex flex-col items-center justify-center pb-12 text-center md:last:col-span-3 lg:last:col-span-1">
                      <ImageWithBlur src={image} alt={name} className="w-12 h-12 rounded-full" />
                      <p className="mt-8 text-sm font-bold text-white md:whitespace-nowrap">
                        {name}
                      </p>
                      <p className="text-sm text-white/60 md:whitespace-nowrap">{firm}</p>
                    </div>
                  );
                })}
              </div>
              <div className="w-full h-[1px] bg-gradient-to-r from-black to-black via-white/40 mt-[100px] lg:mt-[80px]" />
              <SectionTitle
                className="mt-[100px] lg:mt-[100px]"
                align="center"
                title="From our blog"
                text="Explore insights, tips, and updates directly from our team members"
              />
              <div className="flex flex-row w-full mx-auto gap-6 mt-[96px] flex-wrap lg:flex-nowrap">
                {posts.map((post) => {
                  return (
                    <div className="flex w-full mx-auto mt-6 ">
                      <Link key={post._raw.flattenedPath} href={`${post._raw.flattenedPath}`}>
                        <BlogCard
                          tags={post.tags}
                          imageUrl={post.image ?? "/images/blog-images/defaultBlog.png"}
                          title={post.title}
                          subTitle={post.description}
                          author={authors[post.author]}
                          publishDate={post.date}
                        />
                      </Link>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      </Container>
      <CTA />
    </div>
  );
}

function PhotoLabel({ text, className }: { text: string; className: string }) {
  return (
    <div
      className={cn(
        className,
        "bg-gradient-to-r from-black/70 to-black/40 px-4 py-1.5 rounded-[6px] backdrop-blur-md border-[0.75px] border-white/20 min-w-[140px] flex justify-center",
      )}
    >
      <p className="text-xs text-white">{text}</p>
    </div>
  );
}

function Value({ title, text }: { title: string; text: string }) {
  return (
    <div className="flex bg-black p-10">
      <div>
        <div className="flex items-center">
          <h3 className="font-medium">{title}</h3>
        </div>
        <p className="text-white/60 text-sm leading-6 lg:max-w-[4500px] xl:max-w-[280px] pt-2">
          {text}
        </p>
      </div>
    </div>
  );
}
