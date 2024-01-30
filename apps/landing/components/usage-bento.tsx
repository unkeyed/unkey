// import { BillingData } from "@/components/svg/billing-data";

export function UsageBento() {
  return (
    <div className="w-full relative border-[.75px] h-[576px] mt-10 xl:mt-0 w-[700px] rounded-[32px] border-[#ffffff]/20 flex overflow-x-hidden flex justify-center items-start ">
      <div className="absolute top-[-50px] left-[80px] xs:left-auto">
        <BillingItem className="billing-border-link" />
        <BillingItem className="billing-border-link" />
        <BillingItem className="billing-border-link" />
        <BillingItem className="billing-border-link" />
        <BillingItem />
      </div>
      <UsageText />
    </div>
  );
}

export function BillingItem({ className }: { className?: string }) {
  return (
    <div
      className={`flex rounded-xl border-[.75px] w-[440px] usage-item-gradient border-white/20 mt-6 flex justify-between items-center py-[12px] px-[16px] ${className}`}
    >
      <div className="rounded-full bg-gray-500 flex items-center justify-center p-2 border-.75px border-white/20 bg-white/10">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
        >
          <path d="M2.5 4.5H13.5M13.5 4.5L10.5 1.5M13.5 4.5L10.5 7.5" stroke="white" />
          <path d="M13.5 11.5H2.5M2.5 11.5L5.5 8.5M2.5 11.5L5.5 14.5" stroke="white" />
        </svg>
      </div>
      <p className="flex items-center text-sm text-white">
        Andreas
        <span className="ml-2 text-white/40">retrieved API usage data</span>
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
      <div className="flex items-center h-full">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
        >
          <circle cx="8" cy="8" r="5.5" stroke="white" stroke-opacity="0.25" />
          <path d="M8.5 5V8L10.5 9.5" stroke="white" stroke-opacity="0.25" />
        </svg>
        <p className="ml-2 text-sm text-white/20">8 ms</p>
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
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M15.8536 1.85359L14.7047 3.00245C19.3045 3.11116 23 6.87404 23 11.5C23 16.1945 19.1944 20 14.5 20H14V19H14.5C18.6421 19 22 15.6422 22 11.5C22 7.42813 18.755 4.11412 14.71 4.00292L15.8536 5.14648L15.1464 5.85359L13.1464 3.85359L12.7929 3.50004L13.1464 3.14648L15.1464 1.14648L15.8536 1.85359ZM9.5 4.00004C5.35786 4.00004 2 7.3579 2 11.5C2 15.5719 5.24497 18.886 9.29001 18.9972L8.14645 17.8536L8.85355 17.1465L10.8536 19.1465L11.2071 19.5L10.8536 19.8536L8.85355 21.8536L8.14645 21.1465L9.29531 19.9976C4.69545 19.8889 1 16.126 1 11.5C1 6.80562 4.80558 3.00004 9.5 3.00004H10V4.00004H9.5ZM12 8.00004V7.00004H11V8.00004C9.89543 8.00004 9 8.89547 9 10C9 11.1046 9.89543 12 11 12H13C13.5523 12 14 12.4478 14 13C14 13.5523 13.5523 14 13 14H12.5H9.5V15H12V16H13V15C14.1046 15 15 14.1046 15 13C15 11.8955 14.1046 11 13 11H11C10.4477 11 10 10.5523 10 10C10 9.44775 10.4477 9.00004 11 9.00004H11.5H14.5V8.00004H12Z"
            fill="white"
            fill-opacity="0.4"
          />
        </svg>
        <h3 className="ml-4 text-lg font-medium text-white">Usage based billing</h3>
      </div>
      <p className="mt-4 text-white/60 leading-6 max-w-[350px]">
        Commercialise your API, establish transparent billing effortlessly on the Unkey platform for
        efficient and streamlined financial processes.
      </p>
    </div>
  );
}
