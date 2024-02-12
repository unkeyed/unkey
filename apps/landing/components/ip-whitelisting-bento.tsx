import { IpWhitelisting } from "@/components/svg/ip-whitelisting";

export function IpWhitelistingBento() {
  return (
    <div className="w-full mt-5 ip-blur-gradient relative ip-whitelisting-bg-gradient border-[.75px] h-[520px] rounded-[32px] border-[#ffffff]/20 flex overflow-x-hidden">
      <IpWhitelisting className="h-[500px] w-full" />
      <IpWhitelistingText />
    </div>
  );
}

export function IpWhitelistingText() {
  return (
    <div className="flex flex-col text-white absolute left-[20px] sm:left-[40px] xl:left-[40px] bottom-[40px] max-w-[350px]">
      <div className="flex w-full items-center">
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
            d="M19.3732 7.83136L23.3732 3.33136L22.6258 2.66699L18.9781 6.77066L16.8531 4.64562L16.146 5.35273L18.646 7.85273L19.0209 8.22769L19.3732 7.83136ZM3.5 6.0003H3V6.5003V16.5003V17.0003H3.5H6V19.5003V20.0003H6.5H15.5H16V19.5003V17.0003H18.5H19V16.5003V11.0003H18V16.0003H15.5H15V16.5003V19.0003H7V16.5003V16.0003H6.5H4V7.0003H6V12.0003H7V7.0003H9V12.0003H10V7.0003H12V12.0003H13V6.5003V6.0003H12.5H9.5H6.5H3.5ZM15 8.0003V12.0003H16V8.0003H15Z"
            fill="white"
            fill-opacity="0.4"
          />
        </svg>
        <h3 className="text-lg font-medium text-white ml-4">IP Whitelisting</h3>
      </div>
      <p className="mt-4 text-white/60 leading-6">
        Ensure secure access control by allowing only designated IP addresses to interact with your
        system, adding an extra layer of protection.
      </p>
    </div>
  );
}
