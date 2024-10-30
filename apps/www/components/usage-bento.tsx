"use client";
import { AnimatedList } from "@/components/animated-list";

import { UsageSparkles } from "@/components/svg/usage";
export function UsageBento() {
  return (
    <div className="w-full overflow-hidden relative border-[.75px] h-[576px] rounded-[32px] usage-bento-bg-gradient border-[#ffffff]/10">
      <UsageSparkles className="absolute top-0" />
      <div className="relative ">
        <AnimatedList className="w-full">
          <BillingItem icon={<ExchangeIcon />} text="Unkey created API key" latency="3 s" />
          <BillingItem
            icon={<LocationIcon />}
            text="User verified key and logged usage"
            latency="1 s"
          />
          <BillingItem
            icon={<OptionsIcon />}
            text="Andreas enabled automatic billing"
            latency="8 ms"
          />
          <BillingItem icon={<BillingIcon />} text="Unkey sent invoice to customer" latency="1 s" />
          <BillingItem icon={<PaymentsIcon />} text="Andreas collected payments" latency="2 s" />
        </AnimatedList>
      </div>
      <UsageText />
    </div>
  );
}

const ExchangeIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none">
    <path d="M2.5 4.5H13.5M13.5 4.5L10.5 1.5M13.5 4.5L10.5 7.5" stroke="white" />
    <path d="M13.5 11.5H2.5M2.5 11.5L5.5 8.5M2.5 11.5L5.5 14.5" stroke="white" />
  </svg>
);

const OptionsIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none">
    <path d="M5 8.5H9" stroke="white" strokeLinejoin="round" />
    <path d="M5 15.5H13" stroke="white" strokeLinejoin="round" />
    <path d="M12 8.5H19" stroke="white" strokeLinejoin="round" />
    <path d="M16 15.5H19" stroke="white" strokeLinejoin="round" />
    <path
      d="M10.5 10.5C11.6046 10.5 12.5 9.60457 12.5 8.5C12.5 7.39543 11.6046 6.5 10.5 6.5C9.39543 6.5 8.5 7.39543 8.5 8.5C8.5 9.60457 9.39543 10.5 10.5 10.5Z"
      stroke="white"
      strokeLinejoin="round"
    />
    <path
      d="M14.5 17.5C15.6046 17.5 16.5 16.6046 16.5 15.5C16.5 14.3954 15.6046 13.5 14.5 13.5C13.3954 13.5 12.5 14.3954 12.5 15.5C12.5 16.6046 13.3954 17.5 14.5 17.5Z"
      stroke="white"
      strokeLinejoin="round"
    />
  </svg>
);

const LocationIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none">
    <path d="M9 12L11.5 14L15.5 9.5" stroke="white" />
    <path
      d="M5.10608 7.24025L11.1227 4.66171C11.3716 4.55502 11.6397 4.5 11.9105 4.5H12.0895C12.3603 4.5 12.6284 4.55502 12.8773 4.66171L18.8939 7.24025C19.2616 7.39783 19.5 7.75937 19.5 8.1594V8.5L19.0821 11.6346C18.7148 14.3888 17.096 16.819 14.6958 18.2191L12.9319 19.2481C12.649 19.4131 12.3275 19.5 12 19.5C11.6725 19.5 11.351 19.4131 11.0681 19.2481L9.30415 18.2191C6.90403 16.819 5.28517 14.3888 4.91794 11.6346L4.5 8.5V8.1594C4.5 7.75937 4.7384 7.39783 5.10608 7.24025Z"
      stroke="white"
    />
  </svg>
);

const BillingIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none">
    <path
      d="M14.5 9.5H11.5M9.5 15.5H12.5M12.5 15.5H13C13.8284 15.5 14.5 14.8284 14.5 14V14C14.5 13.1716 13.8284 12.5 13 12.5H11C10.1716 12.5 9.5 11.8284 9.5 11V11C9.5 10.1716 10.1716 9.5 11 9.5H11.5M12.5 15.5V17M11.5 9.5V8"
      stroke="white"
    />
    <path
      d="M5.5 18V6C5.5 5.17157 6.17157 4.5 7 4.5H13.8787C14.2765 4.5 14.658 4.65803 14.9393 4.93934L18.0607 8.06066C18.342 8.34196 18.5 8.7235 18.5 9.12132V18C18.5 18.8284 17.8284 19.5 17 19.5H7C6.17157 19.5 5.5 18.8284 5.5 18Z"
      stroke="white"
    />
  </svg>
);

const PaymentsIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none">
    <path
      d="M0.5 12V4C0.5 3.17157 1.17157 2.5 2 2.5H14C14.8284 2.5 15.5 3.17157 15.5 4V12C15.5 12.8284 14.8284 13.5 14 13.5H2C1.17157 13.5 0.5 12.8284 0.5 12Z"
      stroke="white"
    />
    <circle cx="8" cy="8" r="2.25" fill="white" fillOpacity="0.99" />
    <path d="M1 3H3.5C3.5 4.38071 2.38071 5.5 1 5.5V3Z" fill="white" />
    <path d="M1 10.5C2.38071 10.5 3.5 11.6193 3.5 13H1V10.5Z" fill="white" />
    <path d="M12.5 3H15V5.5C13.6193 5.5 12.5 4.38071 12.5 3Z" fill="white" />
    <path d="M12.5 13C12.5 11.6193 13.6193 10.5 15 10.5V13H12.5Z" fill="white" />
  </svg>
);

export function BillingItem({
  className,
  icon,
  text,
  latency,
}: { className?: string; latency: string; icon: React.ReactNode; text: string }) {
  let [first, ...rest] = text.split(" ");
  //@ts-ignore
  rest = rest.join(" ");
  return (
    <div
      className={`flex relative -top-7 left-14 md:left-0 rounded-xl border-[.75px] w-[440px] usage-item-gradient border-white/20 mt-4 md:ml-5 lg:ml-0 flex items-center py-[12px] px-[16px] ${className}`}
    >
      <div className="rounded-full bg-gray-500 flex items-center justify-center h-8 w-8 border-.75px border-white/20 bg-white/10">
        {icon}
      </div>
      <p className="flex items-center ml-6 text-sm text-white">
        {first}
        <span className="ml-2 text-white/40">{rest}</span>
        <svg
          className="inline-flex ml-2"
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
        >
          <path d="M5 13L8 15.5L13.5 8.5M11.5 14L13.5 15.5L19.5 8.5" stroke="#3CEEAE" />
        </svg>
      </p>
      <div className="flex items-center h-full ml-auto">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
        >
          <circle cx="8" cy="8" r="5.5" stroke="white" strokeOpacity="0.25" />
          <path d="M8.5 5V8L10.5 9.5" stroke="white" strokeOpacity="0.25" />
        </svg>
        <p className="ml-2 text-sm text-white/20">{latency}</p>
      </div>
    </div>
  );
}

export function UsageText() {
  return (
    <div className="flex flex-col text-white absolute left-[20px] sm:left-[40px] xl:left-[40px] bottom-[40px] max-w-[3300px]">
      <div className="flex items-center w-full">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
        >
          <path
            fillRule="evenodd"
            clipRule="evenodd"
            d="M15.8536 1.85359L14.7047 3.00245C19.3045 3.11116 23 6.87404 23 11.5C23 16.1945 19.1944 20 14.5 20H14V19H14.5C18.6421 19 22 15.6422 22 11.5C22 7.42813 18.755 4.11412 14.71 4.00292L15.8536 5.14648L15.1464 5.85359L13.1464 3.85359L12.7929 3.50004L13.1464 3.14648L15.1464 1.14648L15.8536 1.85359ZM9.5 4.00004C5.35786 4.00004 2 7.3579 2 11.5C2 15.5719 5.24497 18.886 9.29001 18.9972L8.14645 17.8536L8.85355 17.1465L10.8536 19.1465L11.2071 19.5L10.8536 19.8536L8.85355 21.8536L8.14645 21.1465L9.29531 19.9976C4.69545 19.8889 1 16.126 1 11.5C1 6.80562 4.80558 3.00004 9.5 3.00004H10V4.00004H9.5ZM12 8.00004V7.00004H11V8.00004C9.89543 8.00004 9 8.89547 9 10C9 11.1046 9.89543 12 11 12H13C13.5523 12 14 12.4478 14 13C14 13.5523 13.5523 14 13 14H12.5H9.5V15H12V16H13V15C14.1046 15 15 14.1046 15 13C15 11.8955 14.1046 11 13 11H11C10.4477 11 10 10.5523 10 10C10 9.44775 10.4477 9.00004 11 9.00004H11.5H14.5V8.00004H12Z"
            fill="white"
            fillOpacity="0.4"
          />
        </svg>
        <h3 className="relative z-50 ml-4 text-lg font-medium text-white bg-transparent">
          Monetize your API
        </h3>
      </div>
      <p className="mt-4 text-white/60 leading-6 max-w-[350px]">
        Unkey tracks all user actions in your API, making it straightforward to bill users based on
        their usage.
      </p>
    </div>
  );
}
