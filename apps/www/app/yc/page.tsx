import type React from "react";
import { ContactForm } from "./components/contactForm";
import { DetailsComponent } from "./components/details";

export default function UnkeyYCPage() {
  return (
    <div className="min-h-screen md:pt-20">
      <div className="container mx-auto px-4 md:py-16 ">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-16">
          <DetailsComponent />
          <ContactForm />
        </div>
      </div>
    </div>
  );
}
