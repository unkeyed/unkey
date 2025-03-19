import type React from "react";
import { ContactForm } from "./components/contactForm";
import { DetailsComponent } from "./components/details";
import {
  TopRightShiningLight,
  TopLeftShiningLight,
} from "@/components/svg/background-shiny";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";

export default function UnkeyYCPage() {
  return (
    <>
      <div>
        <TopLeftShiningLight />
      </div>
      <div className="w-full h-full overflow-hidden -z-20">
        <MeteorLinesAngular
          number={1}
          xPos={0}
          speed={10}
          delay={5}
          className="overflow-hidden"
        />
        <MeteorLinesAngular
          number={1}
          xPos={0}
          speed={10}
          delay={0}
          className="overflow-hidden"
        />
        <MeteorLinesAngular
          number={1}
          xPos={100}
          speed={10}
          delay={7}
          className="overflow-hidden md:hidden"
        />
        <MeteorLinesAngular
          number={1}
          xPos={100}
          speed={10}
          delay={2}
          className="overflow-hidden md:hidden"
        />
        <MeteorLinesAngular
          number={1}
          xPos={200}
          speed={10}
          delay={7}
          className="hidden overflow-hidden md:block"
        />
        <MeteorLinesAngular
          number={1}
          xPos={200}
          speed={10}
          delay={2}
          className="hidden overflow-hidden md:block"
        />
        <MeteorLinesAngular
          number={1}
          xPos={400}
          speed={10}
          delay={5}
          className="hidden overflow-hidden lg:block"
        />
        <MeteorLinesAngular
          number={1}
          xPos={400}
          speed={10}
          delay={0}
          className="hidden overflow-hidden lg:block"
        />
      </div>
      <div>
        <TopRightShiningLight />
      </div>

      <div className="min-h-screen md:pt-20 flex justify-center items-center">
        <div className="container px-4 ">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-16">
            <DetailsComponent />
            <ContactForm />
          </div>
        </div>
      </div>
    </>
  );
}
