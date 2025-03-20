import { TopLeftShiningLight, TopRightShiningLight } from "@/components/svg/background-shiny";
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { ContactForm } from "./components/contactForm";
import { DetailsComponent } from "./components/details";

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
          className="overflow-hidden bg-gradient-to-r from-orange-500 to-transparent shadow-[0_0_0_1px_#ffffff10]"
        />
        <MeteorLinesAngular
          number={1}
          xPos={0}
          speed={10}
          delay={0}
          className="overflow-hidden bg-gradient-to-r from-orange-500 to-transparent shadow-[0_0_0_1px_#ffffff10]"
        />
        <MeteorLinesAngular
          number={1}
          xPos={100}
          speed={10}
          delay={7}
          className="overflow-hidden md:hidden bg-gradient-to-r from-orange-500 to-transparent shadow-[0_0_0_1px_#ffffff10]"
        />
        <MeteorLinesAngular
          number={1}
          xPos={100}
          speed={10}
          delay={2}
          className="overflow-hidden md:hidden bg-gradient-to-r from-orange-500 to-transparent shadow-[0_0_0_1px_#ffffff10]"
        />
        <MeteorLinesAngular
          number={1}
          xPos={200}
          speed={10}
          delay={7}
          className="hidden overflow-hidden md:block bg-gradient-to-r from-orange-500 to-transparent shadow-[0_0_0_1px_#ffffff10]"
        />
        <MeteorLinesAngular
          number={1}
          xPos={200}
          speed={10}
          delay={2}
          className="hidden overflow-hidden md:block bg-gradient-to-r from-orange-500 to-transparent shadow-[0_0_0_1px_#ffffff10]"
        />
        <MeteorLinesAngular
          number={1}
          xPos={400}
          speed={10}
          delay={5}
          className="hidden overflow-hidden lg:block bg-gradient-to-r from-orange-500 to-transparent shadow-[0_0_0_1px_#ffffff10]"
        />
        <MeteorLinesAngular
          number={1}
          xPos={400}
          speed={10}
          delay={0}
          className="hidden overflow-hidden lg:block bg-gradient-to-r from-orange-500 to-transparent shadow-[0_0_0_1px_#ffffff10]"
        />
      </div>
      <div>
        <TopRightShiningLight />
      </div>

      <div className="min-h-screen pt-48 flex justify-center items-center">
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
