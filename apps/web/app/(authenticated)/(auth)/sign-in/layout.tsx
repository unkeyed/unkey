import Link from "next/link";
import { Particles } from "@/components/particles";
export default function AuthLayout(props: { children: React.ReactNode }) {
  return (
    <>
      <div className="grid h-screen place-items-center">
        <Particles
          className="absolute inset-0 -z-10 "
          vy={-1}
          quantity={200}
          staticity={500}
          color="#7c3aed"
        />
        <div className="w-full">
          {props.children}
        </div>
      </div>
    </>
  );
}
