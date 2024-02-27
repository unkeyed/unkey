"use client";
import { AnimatedList } from "@/components/animated-list";
import { BillingItem } from "@/components/usage-bento";

const OptionsIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none">
    <path d="M5 8.5H9" stroke="white" stroke-linejoin="round" />
    <path d="M5 15.5H13" stroke="white" stroke-linejoin="round" />
    <path d="M12 8.5H19" stroke="white" stroke-linejoin="round" />
    <path d="M16 15.5H19" stroke="white" stroke-linejoin="round" />
    <path
      d="M10.5 10.5C11.6046 10.5 12.5 9.60457 12.5 8.5C12.5 7.39543 11.6046 6.5 10.5 6.5C9.39543 6.5 8.5 7.39543 8.5 8.5C8.5 9.60457 9.39543 10.5 10.5 10.5Z"
      stroke="white"
      stroke-linejoin="round"
    />
    <path
      d="M14.5 17.5C15.6046 17.5 16.5 16.6046 16.5 15.5C16.5 14.3954 15.6046 13.5 14.5 13.5C13.3954 13.5 12.5 14.3954 12.5 15.5C12.5 16.6046 13.3954 17.5 14.5 17.5Z"
      stroke="white"
      stroke-linejoin="round"
    />
  </svg>
);

export default function SVGTest() {
  return (
    <div className="h-screen flex justify-center items-center">
      <AnimatedList>
        <BillingItem icon={<OptionsIcon />} text="Unkey verified and logged API key" />
        <BillingItem icon={<OptionsIcon />} text="Andreas retrieved API key usage data" />
        <BillingItem icon={<OptionsIcon />} text="Andreas set invoice preferences" />
        <BillingItem icon={<OptionsIcon />} text="Andreas set invoice preferences" />
        <BillingItem icon={<OptionsIcon />} text="Andreas set invoice preferences" />
      </AnimatedList>
    </div>
  );
}
