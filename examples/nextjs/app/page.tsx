import Link from "next/link";

export default function Home() {
  return (
    <main className="flex flex-col items-center justify-between min-h-screen p-24">
      <div className="z-10 items-center justify-between w-full max-w-5xl font-mono text-sm lg:flex">
        <p className="fixed top-0 left-0 flex justify-center w-full pt-8 pb-6 border-b border-gray-300 bg-gradient-to-b from-stone-200 backdrop-blur-2xl dark:border-neutral-800 dark:bg-stone-800/30 dark:from-inherit lg:static lg:w-auto lg:rounded-xl lg:border lg:bg-gray-200 lg:p-4 lg:dark:bg-stone-800/30">
          Get started with&nbsp;
          <Link target="_blank" href="https://unkey.dev" className="font-bold ">
            unkey.dev
          </Link>
        </p>
      </div>

      <div className="relative flex place-items-center">
        Go to
        <Link href="/keys/create" className="px-2 py-1 font-bold hover:underline">
          /keys/create
        </Link>{" "}
        to create a new api key
      </div>

      <div className="grid mb-32 text-center lg:mb-0 lg:grid-cols-4 lg:text-left">
        <a
          href="https://docs.unkey.dev"
          className="px-5 py-4 transition-colors border border-transparent rounded-lg group hover:border-gray-300 hover:bg-gray-100 hover:dark:border-neutral-700 hover:dark:bg-neutral-800/30"
          target="_blank"
          rel="noopener noreferrer"
        >
          <h2 className={"mb-3 text-2xl font-semibold"}>
            Docs{" "}
            <span className="inline-block transition-transform group-hover:translate-x-1 motion-reduce:transform-none">
              -&gt;
            </span>
          </h2>
          <p className={"m-0 max-w-[30ch] text-sm opacity-50"}>
            Find in-depth information about Unkey's features and API.
          </p>
        </a>

        <a
          href="https://github.com/unkeyed/unkey"
          className="px-5 py-4 transition-colors border border-transparent rounded-lg group hover:border-gray-300 hover:bg-gray-100 hover:dark:border-neutral-700 hover:dark:bg-neutral-800 hover:dark:bg-opacity-30"
          target="_blank"
          rel="noopener noreferrer"
        >
          <h2 className={"mb-3 text-2xl font-semibold"}>
            Contribute{" "}
            <span className="inline-block transition-transform group-hover:translate-x-1 motion-reduce:transform-none">
              -&gt;
            </span>
          </h2>
          <p className={"m-0 max-w-[30ch] text-sm opacity-50"}>
            Or check out the code, everything is open source.
          </p>
        </a>

        <a
          href="https://github.com/unkeyed/unkey/tree/main/examples"
          className="px-5 py-4 transition-colors border border-transparent rounded-lg group hover:border-gray-300 hover:bg-gray-100 hover:dark:border-neutral-700 hover:dark:bg-neutral-800/30"
          target="_blank"
          rel="noopener noreferrer"
        >
          <h2 className={"mb-3 text-2xl font-semibold"}>
            Examples{" "}
            <span className="inline-block transition-transform group-hover:translate-x-1 motion-reduce:transform-none">
              -&gt;
            </span>
          </h2>
          <p className={"m-0 max-w-[30ch] text-sm opacity-50"}>Explore the unkey examples</p>
        </a>

        <a
          href="https://github.com/unkeyed/unkey"
          className="px-5 py-4 transition-colors border border-transparent rounded-lg group hover:border-gray-300 hover:bg-gray-100 hover:dark:border-neutral-700 hover:dark:bg-neutral-800/30"
          target="_blank"
          rel="noopener noreferrer"
        >
          <h2 className={"mb-3 text-2xl font-semibold"}>
            Deploy{" "}
            <span className="inline-block transition-transform group-hover:translate-x-1 motion-reduce:transform-none">
              -&gt;
            </span>
          </h2>
          <p className={"m-0 max-w-[30ch] text-sm opacity-50"}>Deploy your own Unkey service</p>
        </a>
      </div>
    </main>
  );
}
