"use client";
import { Particles } from "@/components/particles";
import { addToWaitlist } from "./addToWaitlist";
import { Toaster, toast } from "sonner";


export const runtime = "edge"
export const revalidate = 60

export default function Home() {
  return (
    <main className="relative flex flex-col items-center justify-between min-h-screen px-4 py-24 bg-black bg-gradient-to-t from-violet-400/0 to-violet-400/20">
      <Particles
        className="absolute inset-0"
        vy={-1}
        quantity={50}
        staticity={200}
        color="#7c3aed"
      />

      <div className="z-10 flex flex-col items-center my-4 md:my-8 lg:my-16 xl:my-24">
        <h1 className="py-2 text-6xl font-bold text-center text-transparent xl:text-7xl bg-clip-text bg-gradient-to-t from-violet-100 to-zinc-300 font-display">
          Unkey Your Development
        </h1>

        <p className="max-w-lg mt-8 text-lg font-thin text-center text-gray-300">
          Unkey is an open source API key management platform. Issue, revoke and rotate keys for
          your product in minutes
        </p>

        <span className="px-3 py-1 mt-8 text-sm leading-6 rounded-full bg-black/10 text-violet-200 ring-1 ring-inset ring-violet-100/20">
          Coming soon
        </span>
      </div>

      <form
        className="relative z-10 flex items-center w-full max-w-sm pr-1 mt-8 isolate"
        action={async (data: FormData) => {
          const email = data.get("email");
          if (!email) {
            toast.error("You need to enter an email");
            return;
          }
          toast.promise(addToWaitlist(email as string), {
            loading: "Adding to db",
            success: (data) =>
              `Thank you, you're number ${Intl.NumberFormat().format(data)} on the list`,
            error: "Error",
          });
        }}
      >
        <label htmlFor="email" className="sr-only">
          Email address
        </label>
        <input
          required
          type="email"
          autoComplete="email"
          name="email"
          id="email"
          placeholder="Email address"
          className="peer w-0 flex-auto bg-transparent px-2 py-2.5 text-base  text-white placeholder:text-gray-500 focus:outline-none focus:ring-0 "
        />

        <button
          type="submit"
          className="hidden sm:block group relative isolate flex-none rounded-md py-1.5 text-[0.8125rem]/6 font-semibold text-white pl-2.5 pr-[calc(9/16*1rem)] bg-indigo-300/20"
        >
          Enter Waitlist
          <span aria-hidden="true" className="ml-1">
            &rarr;
          </span>
        </button>
        <button
          type="submit"
          className="sm:hidden group relative isolate flex-none rounded-md p-1.5  text-white  bg-indigo-300/20"
        >
          <kbd aria-hidden="true" className="ml-1">
            ⏎
          </kbd>
        </button>

        <div className="absolute inset-0 transition rounded-lg -z-10 peer-focus:ring-4 peer-focus:ring-indigo-300/15" />
        <div className="absolute inset-0 -z-10 rounded-lg bg-white/2.5 ring-1 ring-white/15 transition peer-focus:ring-indigo-300" />
      </form>
      <Toaster />
    </main>
  );
}
