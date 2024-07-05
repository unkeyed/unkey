import { cn } from "@/lib/utils";
import Image from "next/image";

const logos = [
  {
    name: "Fireworks",
    url: "/images/logo-cloud/fireworks-ai.svg",
  },
  {
    name: "cal.com",
    url: "/images/logo-cloud/calcom.svg",
  },
  {
    name: "Mintlify",
    url: "/images/logo-cloud/mintlify.svg",
  },
];

export function LogoCloudContent() {
  return (
    <div className="w-full flex flex-col items-center">
      <span className={cn("font-mono text-sm md:text-md text-white/50 text-center")}>Powering</span>

      <div className="flex w-full flex-col items-center justify-center px-4 md:px-8">
        <div className="mt-10 grid grid-cols-3 gap-x-6">
          {logos.map((logo, key) => (
            <div key={String(key)} className="relative w-[229px] aspect-[229/36]">
              <Image src={logo.url} alt={`${logo.name}`} fill />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
