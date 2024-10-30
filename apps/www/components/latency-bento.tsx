import map from "../images/map.svg";
import { ImageWithBlur } from "./image-with-blur";

export function LatencyBento() {
  return (
    <div className="w-full relative border-[.75px] h-[576px] rounded-[32px] border-[#ffffff]/10 flex overflow-x-hidden">
      {/* <LatencyMap className="h-[500px] w-full" /> */}
      <ImageWithBlur
        src={map}
        alt="Animated map showing Unkey latency globally"
        className="h-full sm:h-auto"
      />
      <LatencyText />
    </div>
  );
}

export function LatencyText() {
  return (
    <div className="flex flex-col text-white absolute left-[20px] sm:left-[40px] xl:left-[40px] bottom-[40px] max-w-[330px]">
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
            d="M4.26025 9.96741C4.09026 10.6164 3.99976 11.2977 3.99976 12C3.99976 16.0797 7.05355 19.4461 11 19.9381V17.25L9.19996 15.9L8.99996 15.75V15.5V13.5V13.191L9.27635 13.0528L11.2764 12.0528L11.3819 12H11.5H15.5H16V12.5V15V15.1754L15.8904 15.3123L13.9461 17.7427L13.2247 19.9068C17.0615 19.3172 19.9998 16.0017 19.9998 12C19.9998 11.6812 19.9811 11.3669 19.945 11.0582L20.9382 10.9418C20.9789 11.2891 20.9998 11.6422 20.9998 12C20.9998 16.9706 16.9703 21 11.9998 21C7.02919 21 2.99976 16.9706 2.99976 12C2.99976 7.02944 7.02919 3 11.9998 3C13.0689 3 14.0956 3.18664 15.0482 3.52955L14.7095 4.47045C13.864 4.16609 12.9518 4 11.9998 4C11.3408 4 10.7004 4.07968 10.0877 4.22992L11.8123 5.60957L12 5.75969V6V8.5V8.80902L11.7236 8.94721L9.87263 9.87268L8.94716 11.7236L8.80897 12H8.49995H6.49995H6.29284L6.1464 11.8536L4.6464 10.3536L4.26025 9.96741ZM12.14 19.9988C12.0934 19.9996 12.0467 20 12 20V17V16.75L11.8 16.6L9.99996 15.25V13.809L11.618 13H15V14.8246L13.1095 17.1877L13.0538 17.2573L13.0256 17.3419L12.14 19.9988ZM4.61797 8.91091L5.3535 9.64645L6.70706 11H8.19093L9.05274 9.27639L9.12727 9.12732L9.27634 9.05279L11 8.19098V6.24031L8.95124 4.60134C6.99823 5.40693 5.43395 6.96331 4.61797 8.91091ZM18.3735 8.83218L22.3735 4.33218L21.6261 3.66782L17.9783 7.77148L15.8533 5.64645L15.1462 6.35355L17.6462 8.85355L18.0212 9.22852L18.3735 8.83218Z"
            fill="white"
            fillOpacity="0.4"
          />
        </svg>
        <h3 className="ml-4 text-lg font-medium text-white">Global low latency</h3>
      </div>
      <p className="mt-4 leading-6 text-white/60">
        Unkey is fast globally, regardless of which cloud providers you're using or where your users
        are located.
      </p>
    </div>
  );
}
