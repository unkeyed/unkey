import Link from "next/link";

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-between p-24">
      <div className="z-10 w-full max-w-5xl items-center justify-between font-mono text-sm lg:flex">
        <p className="fixed left-0 top-0 flex w-full justify-center border-b border-gray-300 bg-gradient-to-b from-gray-200 pb-6 pt-8 backdrop-blur-2xl dark:border-gray-800 dark:bg-gray-800/30 dark:from-inherit lg:static lg:w-auto lg:rounded-xl lg:border lg:bg-gray-200 lg:p-4 lg:dark:bg-gray-800/30">
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

      <div className="mb-32 grid text-center lg:mb-0 lg:grid-cols-4 lg:text-left">
        <a
          href="https://unkey.dev/docs"
          className="group rounded-lg border border-transparent px-5 py-4 transition-colors hover:border-gray-300 hover:bg-gray-100 hover:dark:border-gray-700 hover:dark:bg-gray-800/30"
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
          className="group rounded-lg border border-transparent px-5 py-4 transition-colors hover:border-gray-300 hover:bg-gray-100 hover:dark:border-gray-700 hover:dark:bg-gray-800 hover:dark:bg-opacity-30"
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
          className="group rounded-lg border border-transparent px-5 py-4 transition-colors hover:border-gray-300 hover:bg-gray-100 hover:dark:border-gray-700 hover:dark:bg-gray-800/30"
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
          className="group rounded-lg border border-transparent px-5 py-4 transition-colors hover:border-gray-300 hover:bg-gray-100 hover:dark:border-gray-700 hover:dark:bg-gray-800/30"
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
