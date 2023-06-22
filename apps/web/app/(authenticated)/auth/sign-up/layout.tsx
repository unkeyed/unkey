import { auth } from "@clerk/nextjs";
import { Particles } from "@/components/particles";
import { redirect } from "next/navigation";

const testimonials: {
  quote: React.ReactNode;
  author: {
    image: string;
    name: string;
    title: string;
  };
}[] = [
  {
    quote: <p>“I build stuff before asking if anyone wants it“</p>,
    author: {
      image: "https://pbs.twimg.com/profile_images/1437670380957835264/gu8S0olw_400x400.jpg",
      name: "Andreas Thomas",
      title: "Random Dude at Upstash",
    },
  },
  {
    quote: (
      <p>
        “When using a 3rd party authentication system, it is most important to replicate a user
        table in your own database!“
      </p>
    ),
    author: {
      image: "https://pbs.twimg.com/profile_images/1613589907133403161/zGtDomUL_400x400.jpg",
      name: "James R Perkins",
      title: "Random Dude at Clerk",
    },
  },
  {
    quote: <p>“I've been passionate about API Keys from a young age actually.“</p>,
    author: {
      image: "https://pbs.twimg.com/profile_images/1629470718630002692/Dax4prIG_400x400.jpg",
      name: "Dom Eccleston",
      title: "Random Dude at Vercel",
    },
  },
];

export default function AuthLayout(props: { children: React.ReactNode }) {
  const _testimonial = testimonials.at(Math.floor(Math.random() * testimonials.length));
  const { userId } = auth();

  if (userId) {
    return redirect("/app/apis");
  }
  return (
    <>
      <div className="relative grid min-h-screen grid-cols-1 overflow-hidden md:grid-cols-3 lg:grid-cols-2">
        <div className="relative flex items-center justify-center bg-white md:bg-black bg-gradient-to-t from-violet-400/0 to-violet-400/20 ">
          <Particles
            className="absolute inset-0"
            vy={-1}
            quantity={50}
            staticity={200}
            color="#7c3aed"
          />
          <div className="lg:pr-4 lg:pt-4 sm:visibility: hidden">
            <div className="lg:max-w-lg">
              <h2 className="text-base font-semibold leading-7 text-violet-500">
                Powerful API Key Management without the hassle
              </h2>
              <p className="mt-2 text-3xl font-bold tracking-tight text-gray-100 sm:text-4xl">
                Save development time
              </p>
              <p className="mt-6 text-lg leading-8 text-gray-400">
                Issue, manage, and revoke keys for any of your APIs in seconds. Comes with included
                per-key ratelimiting and authorization rules.
              </p>

              {/* {testimonial ? (
                <figure className="grid items-center grid-cols-1 mt-16 gap-x-6 gap-y-8 ">
                  <div className="relative col-span-2">
                    <blockquote className="text-xl font-semibold leading-8 text-gray-100 sm:text-2xl sm:leading-9">
                      {testimonial.quote}
                    </blockquote>
                  </div>
                  <div className="flex items-center gap-4">
                    <img
                      className="w-12 h-12 rounded-xl bg-indigo-950 lg:rounded-3xl"
                      src={testimonial.author.image}
                      alt=""
                    />
                    <figcaption className="text-base lg:col-start-1 lg:row-start-3">
                      <div className="font-semibold text-gray-100">{testimonial.author.name}</div>
                      <div className="mt-1 text-gray-500">{testimonial.author.title}</div>
                    </figcaption>
                  </div>
                </figure>
              ) : null} */}
            </div>
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
