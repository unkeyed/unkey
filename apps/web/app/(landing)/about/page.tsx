import { Icons } from "@/components/ui/icons";
import { Eye, Globe, Rocket, User, Users } from "lucide-react";
import React from "react";

export default function About() {
  return (
    <section id="about" className="bg-gray-50">
      <div className="2xl:container 2xl:mx-auto lg:py-16 lg:px-20 md:py-12 md:px-6 py-9 px-4">
      <section className="mb-12">
    <div className="flex flex-wrap">
      <div className="mb-6 w-full shrink-0 grow-0 basis-auto px-3">
        <h2 className="mb-6 text-3xl font-bold">
          What is <u className="text-primary dark:text-primary-400">
            Unkey?</u>
        </h2>
      </div>

      <div className="mb-md-0 mb-6 w-full shrink-0 grow-0 basis-auto px-3">
        <div className="flex flex-wrap justify-center">
          <div className="mb-4 w-full shrink-0 grow-0 basis-auto lg:w-5/12 mx-4 lg:px-3 border border-gray-600 rounded-md p-3">
            <div className="flex">
              <div className="shrink-0">
                  <Globe/>
              </div>
              <div className="ml-4 grow">
                <p className="mb-3 font-bold">Globally Founded, Globally Remote</p>
                <p className="text-neutral-500 dark:text-neutral-300">
                    We are globally remote and were founded that way too. We believe that the best talent is not always in the same place and that we can build a better product by hiring the best talent, no matter where they are.
                </p>
              </div>
            </div>
          </div>

          <div className="mb-4 w-full shrink-0 grow-0 basis-auto lg:w-5/12 lg:px-3 border border-gray-600 rounded-md p-3">
            <div className="flex">
              <div className="shrink-0">
                  <Users/>
              </div>
              <div className="ml-4 grow">
                <p className="mb-3 font-bold">Open Source</p>
                <p className="text-neutral-500 dark:text-neutral-300">
                  Unkey is a fully open source project, we believe that open source leads to better products and better communities. We are committed to building a great open source community around Unkey and providing the ability to self host for those who want it.
                </p>
              </div>
            </div>
          </div>

          <div className="mb-4 w-full shrink-0 grow-0 basis-auto lg:w-5/12 mx-4 lg:px-3 border border-gray-600 rounded-md p-3">
            <div className="flex">
              <div className="shrink-0">
                  <Rocket/>
              </div>
              <div className="ml-4 grow">
                <p className="mb-3 font-bold">Builders, Innovators</p>
                <p className="text-neutral-500 dark:text-neutral-300">
                  We are serial builders who love to innovate. We are always looking for new ways to improve our product and our community. If we aren't working on Unkey, we are probably learning about something new.
                </p>
              </div>
            </div>
          </div>
          <div className="mb-4 w-full shrink-0 grow-0 basis-auto lg:w-5/12 lg:px-3 border border-gray-600 rounded-md p-3">
            <div className="flex">
              <div className="shrink-0">
                  <Eye/>
              </div>
              <div className="ml-4 grow">
                <p className="mb-3 font-bold">Transparent, Open startup</p>
                <p className="text-neutral-500 dark:text-neutral-300">
                    We believe that transparency is the key to building a great company, whether you are a customer, a contributor, or a team member, you should know what we are working on and how Unkey is doing. Our roadmap, metrics, revenue and code are all open for you to see.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <section className="flexmb-32">
  <div className="mb-6 w-full shrink-0 grow-0 basis-auto px-3">
        <h2 className="mb-6 text-3xl font-bold">
          Meet the <u className="text-primary dark:text-primary-400">
            team</u>
        </h2>
        </div>

    <div className="grid gap-x-6 md:grid-cols-2 lg:gap-x-12 max-w-lg md:max-w-3xl mx-auto">
      <div className="mb-6 lg:mb-0">
        <div
          className="block rounded-lg bg-white shadow-[0_2px_15px_-3px_rgba(0,0,0,0.07),0_10px_20px_-2px_rgba(0,0,0,0.04)] dark:bg-neutral-700">
          <div className="relative overflow-hidden bg-cover bg-no-repeat">
            <img src="./andreas.jpeg" className="w-full rounded-t-lg" />
            <a href="https://chronark.com/">
              <div className="absolute top-0 right-0 bottom-0 left-0 h-full w-full overflow-hidden bg-fixed"></div>
            </a>
            <svg className="absolute text-white dark:text-neutral-700 left-0 bottom-0" xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 1440 320">
              <path fill="currentColor"
                d="M0,288L48,272C96,256,192,224,288,197.3C384,171,480,149,576,165.3C672,181,768,235,864,250.7C960,267,1056,245,1152,250.7C1248,256,1344,288,1392,304L1440,320L1440,320L1392,320C1344,320,1248,320,1152,320C1056,320,960,320,864,320C768,320,672,320,576,320C480,320,384,320,288,320C192,320,96,320,48,320L0,320Z">
              </path>
            </svg>
          </div>
          <div className="p-6">
            <h5 className="mb-4 text-lg font-bold">Andreas Thomas</h5>
            <p className="mb-4 text-neutral-500 dark:text-neutral-300">Co Founder</p>
            <ul className="mx-auto flex list-inside justify-center">
            <a href="https://github.com/chronark" className="px-2">
              <Icons.gitHub className="w-6 h-6" />
              </a>
              <a href="https://twitter.com/chronark_" className="px-2">
                <Icons.twitter/>
              </a>
              <a href="mailto:dev@chronark.com" className="px-2">
                <Icons.email/>
              </a>
            </ul>
          </div>
        </div>
      </div>

      <div className="mb-6 lg:mb-0 	">
        <div
          className="block rounded-lg bg-white shadow-[0_2px_15px_-3px_rgba(0,0,0,0.07),0_10px_20px_-2px_rgba(0,0,0,0.04)] dark:bg-neutral-700">
          <div className="relative overflow-hidden bg-cover bg-no-repeat">
            <img src="./james.jpg" className="w-full rounded-t-lg" />
            <a href="https://jamesperkins.dev">
            </a>
            <svg className="absolute text-white dark:text-neutral-700  left-0 bottom-0" xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 1440 320">
              <path fill="currentColor"
                d="M0,96L48,128C96,160,192,224,288,240C384,256,480,224,576,213.3C672,203,768,213,864,202.7C960,192,1056,160,1152,128C1248,96,1344,64,1392,48L1440,32L1440,320L1392,320C1344,320,1248,320,1152,320C1056,320,960,320,864,320C768,320,672,320,576,320C480,320,384,320,288,320C192,320,96,320,48,320L0,320Z">
              </path>
            </svg>
          </div>
          <div className="p-6">
            <h5 className="mb-4 text-lg font-bold">James Perkins</h5>
            <p className="mb-4 text-neutral-500 dark:text-neutral-300">Co Founder</p>
            <ul className="mx-auto flex list-inside justify-center">
              <a href="https://github.com/perkinsjr" className="px-2">
              <Icons.gitHub className="w-6 h-6" />
              </a>
              <a href="https://twitter.com/james_r_perkins" className="px-2">
                <Icons.twitter/>
              </a>
              <a href="mailto:james@jamesperkins.dev" className="px-2">
                <Icons.email/>
              </a>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </section>
  </div>
    </section>
  );
}
