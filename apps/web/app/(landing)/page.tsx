import Link from "next/link";

import { db, schema, sql } from "@unkey/db";
import { Github } from "lucide-react";

export const revalidate = 60;

export default async function LandingPage() {
  const [workspaces, apis, keys, stars] = await Promise.all([
    db
      .select({ count: sql<number>`count(*)` })
      .from(schema.workspaces)
      .then((res) => res.at(0)?.count ?? 0),
    db
      .select({ count: sql<number>`count(*)` })
      .from(schema.apis)
      .then((res) => res.at(0)?.count ?? 0),
    db
      .select({ count: sql<number>`count(*)` })
      .from(schema.keys)
      .then((res) => res.at(0)?.count ?? 0),
    await fetch("https://api.github.com/repos/chronark/unkey", {
      headers: {
        Authorization: `Bearer ${process.env.GITHUB_TOKEN}`,
        "Content-Type": "application/json",
      },
    }).then((res) => res.json()),
  ]);
  return (
    <>
      <div className="overflow-x-hidden bg-gray-50">
        <section className="flex flex-col items-center justify-center min-h-screen pt-12 bg-gray-50 sm:pt-16">
          <div className="px-4 mx-auto max-w-7xl sm:px-6 lg:px-8">
            <div className="max-w-2xl mx-auto text-center">
              {/*<Link className="px-6 text-lg font-bold text-gray-600" href="https://github.com/chronark/unkey">Unkey</Link>*/}
              <h1 className="mt-5 text-4xl font-bold leading-tight text-gray-900 font-display sm:leading-tight sm:text-5xl lg:text-6xl xl:text-7xl lg:leading-tight font-pj">
                Accelerate your API Development
              </h1>

              <div className="relative mt-8">
                <div className="absolute -inset-2">
                  <div className="w-full h-full mx-auto opacity-30 blur-lg filter" />
                </div>
                <div className="relative inline-flex mt-10 group">
                  <div className="absolute transitiona-all duration-1000 opacity-70 -inset-px bg-gradient-to-r from-[#44BCFF] via-[#FF44EC] to-[#FF675E] rounded-lg blur-lg filter group-hover:opacity-100 group-hover:-inset-1 group-hover:duration-200" />
                  <Link
                    href="/auth/sign-up"
                    className="relative inline-flex items-center justify-center px-8 py-4 text-lg font-bold text-white transition-all duration-200 bg-gray-900 rounded-lg font-pj focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-900"
                  >
                    Start Building
                  </Link>
                </div>
              </div>

              <p className="mt-16 text-base font-medium text-gray-700 ">
                Unkey is an open source API Key management solution. It allows you to create, manage
                and validate API Keys for your users. It’s built with security and speed in mind.
              </p>
            </div>
          </div>

          <div className="relative pt-16 overflow-hidden">
            <div className="px-6 mx-auto max-w-7xl lg:px-8">
              <img
                src="/images/landing/app.png"
                alt="App screenshot"
                className="mb-[-12%] rounded-lg shadow-2xl ring-1 ring-gray-900/10"
                width={2432}
                height={1442}
              />
              <div className="relative" aria-hidden="true">
                <div className="absolute -inset-x-20 bottom-0 bg-gradient-to-t from-zinc-50 pt-[7%]" />
              </div>
            </div>
          </div>
        </section>
      </div>

      <section className="py-12 bg-gray-50 sm:py-16 lg:py-20">
        <div className="px-4 mx-auto max-w-7xl sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-3xl font-bold text-gray-900 sm:text-4xl xl:text-5xl font-pj">
              How it works
            </h2>
            <p className="max-w-md mx-auto mt-5 text-base font-normal text-gray-600 font-pj">
              Get started in under five minutes
            </p>
          </div>

          <div className="flex flex-col items-center max-w-md mx-auto mt-8 lg:mt-20 lg:flex-row lg:max-w-none">
            <div className="relative flex-1 w-full overflow-hidden bg-white border border-gray-200 rounded-2xl">
              <div className="py-8 px-9">
                <div className="inline-flex items-center justify-center w-10 h-10 text-base font-bold text-white bg-gray-900 rounded-lg font-pj">
                  1
                </div>
                <p className="mt-5 text-lg font-medium text-gray-900 font-pj">
                  Sign up for your free Unkey account.
                </p>
              </div>
            </div>

            <div className="block -my-1 lg:hidden">
              <svg
                className="w-4 h-auto text-gray-300"
                viewBox="0 0 16 32"
                fill="none"
                stroke="currentColor"
                xmlns="http://www.w3.org/2000/svg"
              >
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(0.83205 0.5547 0.5547 -0.83205 1 21)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(0.83205 0.5547 0.5547 -0.83205 1 14)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(0.83205 0.5547 0.5547 -0.83205 1 7)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(0.83205 0.5547 0.5547 -0.83205 1 0)"
                />
              </svg>
            </div>

            <div className="hidden lg:block lg:-mx-2">
              <svg
                className="w-auto h-4 text-gray-300"
                viewBox="0 0 81 16"
                fill="none"
                stroke="currentColor"
                xmlns="http://www.w3.org/2000/svg"
              >
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 11 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 46 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 81 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 18 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 53 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 25 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 60 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 32 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 67 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 39 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 74 1)"
                />
              </svg>
            </div>

            <div className="relative flex-1 w-full">
              <div className="absolute -inset-4">
                <div
                  className="w-full h-full mx-auto rotate-180 opacity-20 blur-lg filter"
                  style={{
                    background:
                      "linear-gradient(90deg, #44ff9a -0.55%, #44b0ff 22.86%, #8b44ff 48.36%, #ff6644 73.33%, #ebff70 99.34%)",
                  }}
                />
              </div>

              <div className="relative overflow-hidden bg-white border border-gray-200 rounded-2xl">
                <div className="py-8 px-9">
                  <div className="inline-flex items-center justify-center w-10 h-10 text-base font-bold text-white bg-gray-900 rounded-lg font-pj">
                    2
                  </div>
                  <p className="mt-5 text-lg font-medium text-gray-900 font-pj">
                    Create keys in the dashboard, or using the Unkey API
                  </p>
                </div>
              </div>
            </div>

            <div className="hidden lg:block lg:-mx-2">
              <svg
                className="w-auto h-4 text-gray-300"
                viewBox="0 0 81 16"
                fill="none"
                stroke="currentColor"
                xmlns="http://www.w3.org/2000/svg"
              >
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 11 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 46 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 81 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 18 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 53 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 25 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 60 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 32 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 67 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 39 1)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(-0.5547 0.83205 0.83205 0.5547 74 1)"
                />
              </svg>
            </div>

            <div className="block -my-1 lg:hidden">
              <svg
                className="w-4 h-auto text-gray-300"
                viewBox="0 0 16 32"
                fill="none"
                stroke="currentColor"
                xmlns="http://www.w3.org/2000/svg"
              >
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(0.83205 0.5547 0.5547 -0.83205 1 21)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(0.83205 0.5547 0.5547 -0.83205 1 14)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(0.83205 0.5547 0.5547 -0.83205 1 7)"
                />
                <line
                  y1="-0.5"
                  x2="18.0278"
                  y2="-0.5"
                  transform="matrix(0.83205 0.5547 0.5547 -0.83205 1 0)"
                />
              </svg>
            </div>

            <div className="relative flex-1 w-full overflow-hidden bg-white border border-gray-200 rounded-2xl">
              <div className="py-8 px-9">
                <div className="inline-flex items-center justify-center w-10 h-10 text-base font-bold text-white bg-gray-900 rounded-lg font-pj">
                  3
                </div>
                <p className="mt-5 text-lg font-medium text-gray-900 font-pj">
                  Verify your user as part of your API authorization
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section className="py-12 bg-gray-50 sm:py-16 lg:py-20">
        <div className="px-4 mx-auto max-w-7xl sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 md:items-center gap-y-8 md:grid-cols-2 md:gap-x-16">
            <div>
              {/* <img className="w-full max-w-lg mx-auto" src="https://cdn.rareblocks.xyz/collection/clarity/images/features/3/illustration.png" alt="" /> */}
            </div>

            <div className="lg:pr-12">
              <h2 className="text-3xl font-bold text-gray-900 sm:text-4xl xl:text-5xl font-pj">
                Keys where your users are
              </h2>
              <p className="mt-4 text-lg text-gray-700 sm:mt-5 font-pj">
                Your users are everywhere and so is Unkey. Unkey stores keys globally, making each
                request as fast possible regardless of your location.
              </p>
              <div className="relative inline-flex mt-10 group">
                <div className="absolute transitiona-all duration-1000 opacity-70 -inset-px bg-gradient-to-r from-[#44BCFF] via-[#FF44EC] to-[#FF675E] rounded-lg blur-lg filter group-hover:opacity-100 group-hover:-inset-1 group-hover:duration-200" />
                <Link
                  href="https://unkey.dev/auth/sign-up"
                  target="_blank"
                  title=""
                  className="relative inline-flex items-center justify-center px-8 py-4 text-lg font-bold text-white transition-all duration-200 bg-gray-900 rounded-lg font-pj focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-900"
                  role="button"
                >
                  Start Building
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section className="py-12 bg-white sm:py-16 lg:py-20">
        <div className="px-4 mx-auto max-w-7xl sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 md:items-center gap-y-8 md:grid-cols-2 md:gap-x-16">
            <div className="text-center md:text-left lg:pr-16">
              <h2 className="text-3xl font-bold text-gray-900 sm:text-4xl xl:text-5xl font-pj">
                Per-key rate limiting
              </h2>
              <p className="mt-4 text-base text-gray-700 sm:mt-8 font-pj">
                We understand that each user is different, so Unkey gives you the ability to decide
                the rate limits as you issue each key. Giving you complete control while protecting
                your application.
              </p>

              <div className="relative inline-flex mt-10 group">
                <div className="absolute transitiona-all duration-1000 opacity-70 -inset-px bg-gradient-to-r from-[#44BCFF] via-[#FF44EC] to-[#FF675E] rounded-lg blur-lg filter group-hover:opacity-100 group-hover:-inset-1 group-hover:duration-200" />
                <Link
                  href="https://docs.unkey.dev/features/ratelimiting"
                  target="_blank"
                  title=""
                  className="relative inline-flex items-center justify-center px-8 py-4 text-lg font-bold text-white transition-all duration-200 bg-gray-900 rounded-lg font-pj focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-900"
                  role="button"
                >
                  Learn more
                </Link>
              </div>
            </div>

            <div>
              {/* <img className="w-full max-w-lg mx-auto" src="https://cdn.rareblocks.xyz/collection/clarity/images/features/3/illustration.png" alt="" /> */}
            </div>
          </div>
        </div>
      </section>
      <section className="py-12 bg-gray-50 sm:py-16 lg:py-20">
        <div className="px-4 mx-auto max-w-7xl sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 md:items-center gap-y-8 md:grid-cols-2 md:gap-x-16">
            <div>
              {/* <img className="w-full max-w-lg mx-auto" src="https://cdn.rareblocks.xyz/collection/clarity/images/features/3/illustration.png" alt="" /> */}
            </div>

            <div className="lg:pr-12">
              <h2 className="text-3xl font-bold text-gray-900 sm:text-4xl xl:text-5xl font-pj">
                Temporary keys
              </h2>
              <p className="mt-4 text-lg text-gray-700 sm:mt-5 font-pj">
                Want to add a free trial to your API? Unkey allows you to issue temporary keys, once
                the key expires we delete it for you.
              </p>
              <div className="relative inline-flex mt-10 group">
                <div className="absolute transitiona-all duration-1000 opacity-70 -inset-px bg-gradient-to-r from-[#44BCFF] via-[#FF44EC] to-[#FF675E] rounded-lg blur-lg filter group-hover:opacity-100 group-hover:-inset-1 group-hover:duration-200" />
                <Link
                  href="https://docs.unkey.dev/features/temp-keys"
                  target="_blank"
                  title=""
                  className="relative inline-flex items-center justify-center px-8 py-4 text-lg font-bold text-white transition-all duration-200 bg-gray-900 rounded-lg font-pj focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-900"
                  role="button"
                >
                  Learn more
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section className="py-12 bg-white sm:py-16 lg:py-20">
        <div className="px-4 mx-auto max-w-7xl sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 md:items-center gap-y-8 md:grid-cols-2 md:gap-x-16">
            <div className="text-center md:text-left lg:pr-16">
              <h2 className="text-3xl font-bold text-gray-900 sm:text-4xl xl:text-5xl font-pj">
                Usage analytics (Coming Soon)
              </h2>
              <p className="mt-4 text-base text-gray-700 sm:mt-8 font-pj">
                Need to charge a customer on their usage? Want to know who your biggest clients are?
                Unkey provides real time key analytics giving you insights on every user
              </p>
            </div>
            <div>
              {/* <img className="w-full max-w-lg mx-auto" src="https://cdn.rareblocks.xyz/collection/clarity/images/features/3/illustration.png" alt="" /> */}
            </div>
          </div>
        </div>
      </section>
      <section className="py-12 bg-gray-50 sm:py-16 lg:py-20">
        <div className="px-4 mx-auto max-w-7xl sm:px-6 lg:px-8">
          <div className="max-w-2xl mx-auto text-center xl:max-w-4xl">
            <h2 className="text-3xl font-bold text-gray-900 sm:text-4xl xl:text-5xl font-pj">
              We go BRRRRRR
            </h2>
          </div>

          <div className="relative mt-12 lg:mt-20 lg:max-w-5xl lg:mx-auto">
            <div className="absolute -inset-2">
              <div
                className="w-full h-full mx-auto opacity-30 blur-lg filter"
                style={{
                  background:
                    "linear-gradient(90deg, #44ff9a -0.55%, #44b0ff 22.86%, #8b44ff 48.36%, #ff6644 73.33%, #ebff70 99.34%)",
                }}
              />
            </div>

            <div className="relative grid grid-cols-1 px-16 py-12 overflow-hidden text-center bg-white sm:grid-cols-2 gap-y-12 lg:grid-cols-3 rounded-2xl gap-x-20">
              <div className="flex flex-col items-center">
                <p className="text-5xl font-bold text-gray-900 lg:mt-3 lg:order-2 font-pj">
                  {workspaces}
                </p>
                <h3 className="mt-5 text-sm font-bold tracking-widest text-gray-600 uppercase lg:mt-0 lg:order-1 font-pj">
                  Workspaces
                </h3>
              </div>

              <div className="flex flex-col items-center">
                <p className="text-5xl font-bold text-gray-900 lg:mt-3 lg:order-2 font-pj">
                  {apis}
                </p>
                <h3 className="mt-5 text-sm font-bold tracking-widest text-gray-600 uppercase lg:mt-0 lg:order-1 font-pj">
                  APIs
                </h3>
              </div>

              <div className="flex flex-col items-center">
                <p className="text-5xl font-bold text-gray-900 lg:mt-3 lg:order-2 font-pj">
                  {keys}
                </p>
                <h3 className="mt-5 text-sm font-bold tracking-widest text-gray-600 uppercase lg:mt-0 lg:order-1 font-pj">
                  Keys
                </h3>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section className="py-12 bg-white sm:py-16 lg:py-20">
        <div className="px-4 mx-auto bg-white max-w-7xl sm:px-6 lg:px-8">
          <div className="max-w-3xl mx-auto text-center">
            <h1 className="mt-5 text-4xl font-bold leading-tight text-gray-900 font-display sm:leading-tight sm:text-5xl lg:text-6xl xl:text-7xl lg:leading-tight">
              Proudly Open Source
            </h1>

            <p className="mt-8 text-base font-medium text-gray-700 ">
              Our source code is available on GitHub - feel free to review, contribute and share
              with your friends.
            </p>
            <div className="flex items-center justify-center py-10">
              <a href="https://github.com/chronark/unkey" target="_blank" rel="noreferrer">
                <div className="flex items-center">
                  <div className="flex items-center h-10 p-4 space-x-2 bg-gray-800 border border-gray-600 rounded-md">
                    <Github className="w-5 h-5 text-white" />
                    <p className="font-medium text-white">
                      {Intl.NumberFormat().format(stars.stargazers_count || 0)} stars
                    </p>
                    <p />
                  </div>
                </div>
              </a>
            </div>
          </div>
        </div>
      </section>
      <footer className="py-4 bg-white ">
        <div className="px-4 mx-auto max-w-7xl sm:px-6 lg:px-8">
          <hr className="mt-16 border-gray-200" />

          <div className="mt-8 sm:flex sm:items-center sm:justify-between">
            <ul className="flex items-center justify-start space-x-3 sm:order-2 sm:justify-end">
              <li>
                <Link
                  href="https://github.com/chronark/unkey"
                  target="_blank"
                  title=""
                  className="inline-flex items-center justify-center w-10 h-10 text-gray-900 transition-all duration-200 rounded-full hover:bg-gray-100 focus:outline-none focus:bg-gray-200 focus:ring-2 focus:ring-offset-2 focus:ring-gray-200"
                  rel="noopener"
                >
                  <Github className="w-6 h-6" />
                </Link>
              </li>
            </ul>

            <p className="mt-8 text-sm font-normal text-gray-600 font-pj sm:order-1 sm:mt-0">
              © Copyright {new Date().getUTCFullYear()}, All Rights Reserved
            </p>
          </div>
        </div>
      </footer>
    </>
  );
}
