import { auth } from "@clerk/nextjs";
import { redirect } from "next/navigation";
import { Particles } from "@/components/dashboard/particles";

export default function AuthLayout(props: { children: React.ReactNode }) {
  const { userId } = auth();

  if (userId) {
    return redirect("/app/apis");
  }
  return (
    <>
      <div className="relative grid min-h-screen grid-cols-1 overflow-hidden md:grid-cols-3 lg:grid-cols-2 divide-x">
        <div className="container absolute flex items-center col-span-1 -translate-y-1/2 top-1/2 md:static md:top-0 md:col-span-2 md:flex md:translate-y-0 lg:col-span-1">
          {props.children}
        </div>
        <div className="hidden relative md:flex items-center justify-center bg-white md:bg-black h-screen bg-gradient-to-t from-violet-400/0 to-violet-400/20 ">
          <Particles
            className="absolute inset-0 h-screen"
            vy={-1}
            quantity={80}
            staticity={300}
            color="#7c3aed"
          />
        </div>
      </div>
    </>
  );
}
