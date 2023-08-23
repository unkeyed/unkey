import { Particles } from "@/components/dashboard/particles";
import { auth } from "@clerk/nextjs";
import { redirect } from "next/navigation";

const features: {
  title: string;
  description: string;
}[] = [
    {
      title: "Save development time",
      description: "Issue, manage, and revoke keys for your APIs in seconds with built in analytics."
    },
    {
      title: "Globally distributed",
      description: "Unkey Globally distrubtes keys in 35+ locations, making it fast for every user."
    },
    {
      title: "Features for any use case",
      description: "Each key has unique settings such as rate limiting, expiration, and limited uses."
    },
  ];

export default function AuthLayout(props: { children: React.ReactNode }) {

  const { userId } = auth();

  if (userId) {
    return redirect("/app/apis");
  }
  return (
    <>
      <div className="relative grid min-h-screen grid-cols-1 overflow-hidden md:grid-cols-3 lg:grid-cols-2">
        <div className="relative flex items-center justify-center bg-white md:bg-black">
          <div className="lg:pr-4 lg:pt-4 hidden md:block">
            {
              features.map((feature, index) => (
                <div key={index} className="mb-8 lg:max-w-lg">
                  <h3 className="my-2 text-3xl font-bold tracking-tight text-gray-100 sm:text-4xl">
                    {feature.title}
                  </h3>
                  <p className="text-lg leading-8 text-gray-400">
                    {feature.description}
                  </p>
                </div>
              ))
            }
          </div>
        </div>
        <div className="absolute inset-0 bg-gradient-to-t from-background to-background/60 md:hidden" />
        <div className="container absolute flex items-center col-span-1 -translate-y-1/2 top-1/2 md:static md:top-0 md:col-span-2 md:flex md:translate-y-0 lg:col-span-1">
          {props.children}
        </div>
      </div>
    </>
  );
}
