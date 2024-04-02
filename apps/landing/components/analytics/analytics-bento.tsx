"use client";
import { cn } from "@/lib/utils";
import { motion } from "framer-motion";
import { Wand2 } from "lucide-react";
import { Highlight, type PrismTheme } from "prism-react-renderer";
import { useState } from "react";
import { PrimaryButton } from "../button";
import { AnalyticsStars } from "../svg/analytics-stars";
import { WebAppLight } from "../svg/web-app-light";

export const theme = {
  plain: {
    color: "#F8F8F2",
    backgroundColor: "#282A36",
  },
  styles: [
    {
      types: ["string"],
      style: {
        color: "#3CEEAE",
      },
    },
    {
      types: ["string-property"],
      style: {
        color: "#9D72FF",
      },
    },
    {
      types: ["number"],
      style: {
        color: "#FB3186",
      },
    },
  ],
} satisfies PrismTheme;

const codeBlock = `curl --request GET \\
    --url https://api.unkey.dev/v1/keys.getKey \\
    --header 'Authorization: <authorization>'
    {    
      "apiId": "api_1234",
      "createdAt": 123,
      "deletedAt": 123,
      "expires": 123,
      "id": "key_1234",
      "meta": {
        "roles": [
          "admin",
          "user"
        ],
        "stripeCustomerId": "cus_1234"
      }
    }
`;

export function AnalyticsBento() {
  const [showApi, toggleShowApi] = useState(false);

  return (
    <div className="relative flex justify-center w-full">
      <div className="absolute">
        <PrimaryButton
          label="Show API code"
          IconLeft={Wand2}
          onClick={() => toggleShowApi(!showApi)}
        />
      </div>

      <div className="relative mt-[80px] w-full h-[640px] analytics-linear-gradient flex justify-center xl:justify-start items-end border rounded-3xl border border-white/10 relative">
        {/* TODO: horizontal scroll */}
        <LightSvg className="absolute hidden md:flex top-[-180px] left:0 lg:left-[300px] z-50 pointer-events-none" />
        <AnalyticsStars className="w-[120px] shrink-0 hidden md:flex" />
        {showApi ? <AnalyticsApiView /> : <AnalyticsWebAppView />}
        <BentoText />
      </div>
    </div>
  );
}

function AnalyticsApiView() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.2, ease: "easeInOut" }}
      whileInView="visible"
      className="w-full overflow-x-hidden"
    >
      <div className="w-full bg-black bg-opacity-02 lg:w-[1180px] lg:rounded-3xl analytics-background-gradient xxl:mr-10 overflow-x-hidden overflow-y-hidden border-white/10 border border-b-0 border-l-0  border-r-0 flex-col md:flex-row relative rounded-tl-3xl h-[600px] xl:h-[576px] flex">
        <div className="flex flex-col w-[216px] text-white text-sm pt-6 px-4 font-mono md:border-r md:border-white/5">
          <div className="flex items-center cursor-pointer bg-white/5 py-1 px-2 rounded-lg w-[184px]">
            <TerminalIcon className="w-6 h-6" />
            <div className="ml-3">cURL</div>
          </div>
        </div>
        <div className="flex w-full pt-4 pl-8 font-mono text-xs text-white sm:text-sm">
          <Editor theme={theme} codeBlock={codeBlock} language="tsx" />
        </div>
      </div>
    </motion.div>
  );
}

function AnalyticsWebAppView() {
  function Tab({
    backgroundColor,
    text,
    icon,
    light = false,
  }: { backgroundColor: string; text: string; icon: React.ReactNode; light?: boolean }) {
    return (
      <div
        className={cn("flex text-white px-2 py-2 rounded-lg", {
          "bg-white/10": light,
        })}
      >
        <div
          style={{ background: backgroundColor }}
          className="rounded-lg mr-3.5 w-[20px] h-[20px]"
        >
          {icon}
        </div>
        {text}
      </div>
    );
  }

  const icons = {
    sun: (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        viewBox="0 0 20 20"
        fill="none"
      >
        <path
          d="M7.70801 13.5415L7.13526 12.9165C6.70932 12.3318 6.45801 11.6118 6.45801 10.8332C6.45801 8.87717 8.04367 7.2915 9.99967 7.2915C11.9557 7.2915 13.5413 8.87717 13.5413 10.8332C13.5413 11.6118 13.29 12.3318 12.8641 12.9165L12.2913 13.5415"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M3.9502 13.5415H16.0418"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M5.61719 16.0415H14.3755"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M10 3.9585V4.37516"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M13.0208 4.76758L12.8125 5.12842"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M15.2319 6.979L14.8711 7.18734"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M16.0417 10H15.625"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M4.37467 10H3.95801"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M5.12842 7.18734L4.76758 6.979"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M7.18783 5.12843L6.97949 4.76758"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    ),
    translate: (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="24"
        height="20"
        viewBox="0 0 24 20"
        fill="none"
      >
        <path
          d="M12.75 16.0417L16 10.625L19.25 16.0417"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path d="M14 14.375H18" stroke="white" stroke-linecap="round" stroke-linejoin="round" />
        <path d="M5.75 6.625H14.25" stroke="white" stroke-linecap="round" stroke-linejoin="round" />
        <path
          d="M10 6.41683V4.9585"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M12.25 6.625C12.25 6.625 12.25 8.70833 10.25 10.375C8.25 12.0417 5.75 12.0417 5.75 12.0417"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M13.25 12.0417C13.25 12.0417 10.75 12.0417 8.75 10.375C8.34551 10.0379 7.75 9.125 7.75 9.125"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    ),
    analytics: (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        viewBox="0 0 20 20"
        fill="none"
      >
        <path
          d="M8.74967 13.5418H5.62467C5.18265 13.5418 4.75872 13.3662 4.44616 13.0537C4.1336 12.7411 3.95801 12.3172 3.95801 11.8752V5.62516C3.95801 5.18314 4.1336 4.75921 4.44616 4.44665C4.75872 4.13409 5.18265 3.9585 5.62467 3.9585H14.3747C14.8167 3.9585 15.2406 4.13409 15.5532 4.44665C15.8657 4.75921 16.0413 5.18314 16.0413 5.62516V11.8752C16.0413 12.3172 15.8657 12.7411 15.5532 13.0537C15.2406 13.3662 14.8167 13.5418 14.3747 13.5418H11.2497M8.74967 13.5418L7.29134 16.0418M8.74967 13.5418H11.2497M11.2497 13.5418L12.708 16.0418M7.29134 10.2085V7.29183M9.99967 10.2085V8.12516M12.708 10.2085V8.9585"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    ),
    crypto: (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        viewBox="0 0 20 20"
        fill="none"
      >
        <path
          d="M7.29199 7.7085V5.41683M7.29199 5.41683C7.29199 6.22225 9.25074 6.87516 11.667 6.87516C14.0832 6.87516 16.042 6.22225 16.042 5.41683M7.29199 5.41683C7.29199 4.61141 9.25074 3.9585 11.667 3.9585C14.0832 3.9585 16.042 4.61141 16.042 5.41683M16.042 5.41683V8.75016C16.042 9.1105 15.65 9.44025 15.0003 9.69475"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M12.708 11.2498V14.5832C12.708 15.3886 10.7493 16.0415 8.33301 16.0415C5.91677 16.0415 3.95801 15.3886 3.95801 14.5832V11.2498M12.708 11.2498C12.708 12.0553 10.7493 12.7082 8.33301 12.7082C5.91677 12.7082 3.95801 12.0553 3.95801 11.2498M12.708 11.2498C12.708 10.4444 10.7493 9.7915 8.33301 9.7915C5.91677 9.7915 3.95801 10.4444 3.95801 11.2498"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    ),
    bio: (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        viewBox="0 0 20 20"
        fill="none"
      >
        <path
          d="M14.3746 3.9585C14.3746 3.9585 14.9996 10.0002 9.99963 10.0002C4.99961 10.0002 5.62461 16.0418 5.62461 16.0418"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M10 10C15 10 14.375 16.0417 14.375 16.0417"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M5.62501 3.9585C5.62501 3.9585 5.38197 6.76157 6.87523 8.54183"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M5.83301 4.7915H14.1663"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M6.25 7.2915H13.75"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M6.25 12.7085H13.75"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M6.25 15.2085H13.75"
          stroke="white"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    ),
  };
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.2, ease: "easeInOut" }}
      whileInView="visible"
      className="w-full overflow-x-hidden"
    >
      <div className="w-full bg-[#000000] lg:w-[1180px] lg:rounded-3xl relative flex-wrap md:flex-nowrap cursor-default analytics-background-gradient select-none bg-opacity-02 xxl:mr-10 overflow-x-hidden overflow-y-hidden border-white/10 border border-b-0 border-l-0  border-r-0 flex-col md:flex-row relative rounded-tl-3xl h-[600px] xl:h-[576px] flex">
        <WebAppLight className="absolute top-[-100px] left-[40px]" />
        <div className="flex flex-col w-[216px] h-full text-white/20  text-xs pt-6 px-4 md:border-r md:border-white/5">
          <div className="flex justify-between items-center w-[160px]">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="30"
              height="30"
              viewBox="0 0 30 30"
              fill="none"
            >
              <circle cx="15" cy="15" r="14" fill="white" fill-opacity="0.2" />
              <circle
                cx="15"
                cy="15"
                r="14.375"
                stroke="white"
                stroke-opacity="0.3"
                stroke-width="0.75"
              />
              <path
                fill-rule="evenodd"
                clip-rule="evenodd"
                d="M15 24C19.9706 24 24 19.9706 24 15C24 10.0294 19.9706 6 15 6C10.0294 6 6 10.0294 6 15C6 19.9706 10.0294 24 15 24ZM15.0002 10.0001L10.0002 15L15.0001 20L20.0002 15L15.0002 10.0001Z"
                fill="white"
              />
            </svg>
            <p>Acme Co</p>
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
                d="M14.1465 9.85363L12 7.70718L9.85359 9.85363L9.14648 9.14652L11.6465 6.64652L12 6.29297L12.3536 6.64652L14.8536 9.14652L14.1465 9.85363ZM9.85359 14.1465L12 16.293L14.1465 14.1465L14.8536 14.8536L12.3536 17.3536L12 17.7072L11.6465 17.3536L9.14648 14.8536L9.85359 14.1465Z"
                fill="white"
                fill-opacity="0.4"
              />
            </svg>
          </div>
          <p className="my-6">General</p>
          <div>
            <p className="flex">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                className="mr-3"
              >
                <rect
                  x="4.5"
                  y="4.5"
                  width="15"
                  height="15"
                  rx="3.5"
                  stroke="white"
                  stroke-opacity="0.2"
                />
                <path
                  d="M10.5 9.5L8 12L10.5 14.5M13.5 9.5L16 12L13.5 14.5"
                  stroke="white"
                  stroke-opacity="0.2"
                />
              </svg>
              APIs
            </p>
            <p className="flex">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                className="mr-3"
              >
                <path
                  d="M5.62117 14.9627L6.72197 15.1351C7.53458 15.2623 8.11491 16.0066 8.05506 16.8451L7.97396 17.9816C7.95034 18.3127 8.12672 18.6244 8.41885 18.7686L9.23303 19.1697C9.52516 19.3139 9.87399 19.2599 10.1126 19.0352L10.9307 18.262C11.5339 17.6917 12.4646 17.6917 13.0685 18.262L13.8866 19.0352C14.1252 19.2608 14.4733 19.3139 14.7662 19.1697L15.5819 18.7678C15.8733 18.6244 16.0489 18.3135 16.0253 17.9833L15.9441 16.8451C15.8843 16.0066 16.4646 15.2623 17.2772 15.1351L18.378 14.9627C18.6985 14.9128 18.9568 14.6671 19.0292 14.3433L19.23 13.4428C19.3025 13.119 19.1741 12.7831 18.9064 12.5962L17.9875 11.9526C17.3095 11.4774 17.1024 10.5495 17.5119 9.82051L18.067 8.83299C18.2284 8.54543 18.2017 8.18538 17.9993 7.92602L17.4363 7.2035C17.2339 6.94413 16.8969 6.83701 16.5867 6.93447L15.5221 7.26794C14.7355 7.51441 13.8969 7.1012 13.5945 6.31908L13.1866 5.26148C13.0669 4.95218 12.7748 4.7492 12.4496 4.75L11.5472 4.75242C11.222 4.75322 10.9307 4.95782 10.8126 5.26793L10.4149 6.31344C10.1157 7.1004 9.27319 7.51683 8.4842 7.26874L7.37553 6.92078C7.0645 6.82251 6.72591 6.93044 6.52355 7.19142L5.96448 7.91474C5.76212 8.17652 5.73771 8.53738 5.90228 8.82493L6.47 9.81487C6.88812 10.5446 6.68339 11.4814 6.00149 11.9591L5.0936 12.5954C4.82588 12.7831 4.69754 13.119 4.76998 13.442L4.97077 14.3425C5.04242 14.6671 5.30069 14.9128 5.62117 14.9627Z"
                  stroke="white"
                  stroke-opacity="0.2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <path
                  d="M13.5911 10.4089C14.4696 11.2875 14.4696 12.7125 13.5911 13.5911C12.7125 14.4696 11.2875 14.4696 10.4089 13.5911C9.53036 12.7125 9.53036 11.2875 10.4089 10.4089C11.2875 9.53036 12.7125 9.53036 13.5911 10.4089Z"
                  stroke="white"
                  stroke-opacity="0.2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
              Settings
            </p>
            <p className="flex">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                className="mr-3"
              >
                <path
                  fill-rule="evenodd"
                  clip-rule="evenodd"
                  d="M9.32282 6.00095C9.30274 6.00016 9.27513 6.00007 9.21938 6.00007H6.9C6.47171 6.00007 6.18056 6.00046 5.95552 6.01885C5.73631 6.03676 5.62421 6.06922 5.54601 6.10906C5.35785 6.20494 5.20487 6.35792 5.10899 6.54608C5.06915 6.62428 5.03669 6.73638 5.01878 6.95559C5.00039 7.18063 5 7.47178 5 7.90007V14.1001C5 14.5284 5.00039 14.8195 5.01878 15.0446C5.03669 15.2638 5.06915 15.3759 5.10899 15.4541C5.20487 15.6422 5.35785 15.7952 5.54601 15.8911C5.62421 15.9309 5.73631 15.9634 5.95552 15.9813C6.18056 15.9997 6.47171 16.0001 6.9 16.0001H9.21938L9.24102 16H9.24103C9.32199 15.9999 9.42397 15.9996 9.52565 16.0205C9.61392 16.0386 9.69932 16.0686 9.77956 16.1096C9.872 16.1568 9.95148 16.2207 10.0146 16.2714L10.0315 16.2849L11 17.0598V7.34038L9.40678 6.06581C9.36325 6.03098 9.34163 6.0138 9.32546 6.00188L9.32428 6.00101L9.32282 6.00095ZM13.9685 16.2849L13 17.0598V7.34038L14.5932 6.06581C14.6368 6.03098 14.6584 6.0138 14.6745 6.00188L14.6757 6.00101L14.6772 6.00095C14.6973 6.00016 14.7249 6.00007 14.7806 6.00007H17.1C17.5283 6.00007 17.8194 6.00046 18.0445 6.01885C18.2637 6.03676 18.3758 6.06922 18.454 6.10906C18.6422 6.20494 18.7951 6.35792 18.891 6.54608C18.9309 6.62428 18.9633 6.73638 18.9812 6.95559C18.9996 7.18063 19 7.47178 19 7.90007V14.1001C19 14.5284 18.9996 14.8195 18.9812 15.0446C18.9633 15.2638 18.9309 15.3759 18.891 15.4541C18.7951 15.6422 18.6422 15.7952 18.454 15.8911C18.3758 15.9309 18.2637 15.9634 18.0445 15.9813C17.8194 15.9997 17.5283 16.0001 17.1 16.0001H14.7806L14.759 16H14.759C14.678 15.9999 14.576 15.9996 14.4743 16.0205C14.3861 16.0386 14.3007 16.0686 14.2204 16.1096C14.128 16.1568 14.0485 16.2207 13.9854 16.2714L13.9685 16.2849ZM9.24102 5.00004C9.32199 4.99985 9.42397 4.99962 9.52565 5.02049C9.61392 5.0386 9.69932 5.06856 9.77956 5.10955C9.872 5.15678 9.95149 5.22067 10.0146 5.27139L10.0315 5.28494L12 6.85976L13.9685 5.28494L13.9854 5.27139L13.9854 5.27138C14.0485 5.22066 14.128 5.15678 14.2204 5.10955C14.3007 5.06856 14.3861 5.0386 14.4743 5.02049C14.576 4.99962 14.678 4.99985 14.759 5.00004L14.7806 5.00007H17.1L17.1207 5.00007C17.5231 5.00006 17.8553 5.00006 18.1259 5.02217C18.407 5.04513 18.6653 5.09441 18.908 5.21806C19.2843 5.4098 19.5903 5.71576 19.782 6.09209C19.9057 6.33476 19.9549 6.59311 19.9779 6.87416C20 7.14475 20 7.47693 20 7.87941V7.90007V14.1001V14.1207C20 14.5232 20 14.8554 19.9779 15.126C19.9549 15.407 19.9057 15.6654 19.782 15.9081C19.5903 16.2844 19.2843 16.5903 18.908 16.7821C18.6653 16.9057 18.407 16.955 18.1259 16.978C17.8553 17.0001 17.5231 17.0001 17.1207 17.0001H17.1H14.7806C14.7249 17.0001 14.6973 17.0002 14.6772 17.001L14.6757 17.001L14.6745 17.0019C14.6584 17.0138 14.6368 17.031 14.5932 17.0658L12.3123 18.8905L12 19.1404L11.6877 18.8905L9.40678 17.0658C9.36325 17.031 9.34163 17.0138 9.32546 17.0019L9.32428 17.001L9.32282 17.001C9.30274 17.0002 9.27513 17.0001 9.21938 17.0001H6.9H6.87934C6.47686 17.0001 6.14468 17.0001 5.87409 16.978C5.59304 16.955 5.33469 16.9057 5.09202 16.7821C4.7157 16.5903 4.40973 16.2844 4.21799 15.9081C4.09434 15.6654 4.04506 15.407 4.0221 15.126C3.99999 14.8554 3.99999 14.5232 4 14.1207V14.1207V14.1001V7.90007V7.87942V7.87941C3.99999 7.47693 3.99999 7.14475 4.0221 6.87416C4.04506 6.59311 4.09434 6.33476 4.21799 6.09209C4.40973 5.71576 4.7157 5.4098 5.09202 5.21806C5.33469 5.09441 5.59304 5.04513 5.87409 5.02217C6.14469 5.00006 6.47687 5.00006 6.87935 5.00007L6.9 5.00007H9.21938L9.24102 5.00004Z"
                  fill="white"
                  fill-opacity="0.2"
                />
              </svg>
              Docs
            </p>
          </div>
          <p className="mt-8">Your APIs</p>
          <div className="mt-4">
            <Tab backgroundColor="#6E56CF" light text="QuantumWeather" icon={icons.sun} />
            <Tab backgroundColor="#4CBBA5" text="StellarTranslate" icon={icons.translate} />
            <Tab backgroundColor="#978365" text="NebulaAnalytics" icon={icons.analytics} />
            <Tab backgroundColor="#00A2C7" text="CryptoSentiment" icon={icons.crypto} />
            <Tab backgroundColor="#8DB654" text="BioSyncHealth" icon={icons.bio} />
          </div>
        </div>
        <div className="text-white pt-4 pl-8 flex w-full px-[40px] flex-col">
          <div className="flex justify-between w-full h-[40px] items-center">
            <div className="flex items-center text-lg font-medium min-w-[400px]">
              <div className="bg-[#6E56CF] rounded-lg mr-3.5 w-[40px] h-[40px] rounded-[9px]">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="40"
                  height="40"
                  viewBox="0 0 40 40"
                  fill="none"
                >
                  <path
                    d="M15.417 27.0835L14.2715 25.8335C13.4196 24.6642 12.917 23.2242 12.917 21.6668C12.917 17.7548 16.0883 14.5835 20.0003 14.5835C23.9123 14.5835 27.0837 17.7548 27.0837 21.6668C27.0837 23.2242 26.581 24.6642 25.7292 25.8335L24.5837 27.0835"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M7.90039 27.0835H32.0837"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M11.2334 32.0835H28.75"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M20 7.9165V8.74984"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M26.0417 9.53564L25.625 10.2573"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M30.4648 13.9585L29.7432 14.3752"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M32.0833 20H31.25"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M8.75033 20H7.91699"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M10.2578 14.3752L9.53613 13.9585"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M14.3747 10.2573L13.958 9.53564"
                    stroke="white"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </div>
              QuantumWeather
            </div>
            <div className="flex items-center">
              <div className="flex items-center gap-2 px-3 py-1 font-mono text-xs rounded-md rounded-lg bg-white/5 font-sm text-white/40">
                api_UNWrXjYp6AF2h7Nx
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
                    d="M13 5.00002C13.4886 5.00002 13.6599 5.00244 13.7927 5.02884C14.3877 5.1472 14.8528 5.61235 14.9712 6.20738C14.9976 6.34011 15 6.5114 15 7.00002H16L16 6.94215V6.94213C16.0001 6.53333 16.0002 6.25469 15.952 6.01229C15.7547 5.02057 14.9795 4.24532 13.9877 4.04806C13.7453 3.99984 13.4667 3.99991 13.0579 4.00001L13 4.00002H7.70002H7.67861C7.13672 4.00001 6.69965 4.00001 6.34571 4.02893C5.98128 4.0587 5.66119 4.12161 5.36504 4.2725C4.89464 4.51219 4.51219 4.89464 4.2725 5.36504C4.12161 5.66119 4.0587 5.98128 4.02893 6.34571C4.00001 6.69965 4.00001 7.13672 4.00002 7.67862V7.70002V13L4.00001 13.0579C3.99991 13.4667 3.99984 13.7453 4.04806 13.9877C4.24532 14.9795 5.02057 15.7547 6.01229 15.952C6.25469 16.0002 6.53333 16.0001 6.94213 16H6.94215L7.00002 16V15C6.5114 15 6.34011 14.9976 6.20738 14.9712C5.61235 14.8528 5.1472 14.3877 5.02884 13.7927C5.00244 13.6599 5.00002 13.4886 5.00002 13V7.70002C5.00002 7.13172 5.00041 6.73556 5.02561 6.42714C5.05033 6.12455 5.09642 5.95071 5.16351 5.81903C5.30732 5.53679 5.53679 5.30732 5.81903 5.16351C5.95071 5.09642 6.12455 5.05033 6.42714 5.02561C6.73556 5.00041 7.13172 5.00002 7.70002 5.00002H13ZM11.7 8.00002H11.6786C11.1367 8.00001 10.6996 8.00001 10.3457 8.02893C9.98128 8.0587 9.66119 8.12161 9.36504 8.2725C8.89464 8.51219 8.51219 8.89464 8.2725 9.36504C8.12161 9.66119 8.0587 9.98128 8.02893 10.3457C8.00001 10.6996 8.00001 11.1367 8.00002 11.6786V11.7V16.3V16.3214C8.00001 16.8633 8.00001 17.3004 8.02893 17.6543C8.0587 18.0188 8.12161 18.3388 8.2725 18.635C8.51219 19.1054 8.89464 19.4879 9.36504 19.7275C9.66119 19.8784 9.98128 19.9413 10.3457 19.9711C10.6996 20 11.1366 20 11.6785 20H11.6786H11.7H16.3H16.3214H16.3216C16.8634 20 17.3004 20 17.6543 19.9711C18.0188 19.9413 18.3388 19.8784 18.635 19.7275C19.1054 19.4879 19.4879 19.1054 19.7275 18.635C19.8784 18.3388 19.9413 18.0188 19.9711 17.6543C20 17.3004 20 16.8634 20 16.3216V16.3214V16.3V11.7V11.6786V11.6785C20 11.1366 20 10.6996 19.9711 10.3457C19.9413 9.98128 19.8784 9.66119 19.7275 9.36504C19.4879 8.89464 19.1054 8.51219 18.635 8.2725C18.3388 8.12161 18.0188 8.0587 17.6543 8.02893C17.3004 8.00001 16.8633 8.00001 16.3214 8.00002H16.3H11.7ZM9.81903 9.16351C9.95071 9.09642 10.1246 9.05033 10.4271 9.02561C10.7356 9.00041 11.1317 9.00002 11.7 9.00002H16.3C16.8683 9.00002 17.2645 9.00041 17.5729 9.02561C17.8755 9.05033 18.0493 9.09642 18.181 9.16351C18.4632 9.30732 18.6927 9.53679 18.8365 9.81903C18.9036 9.95071 18.9497 10.1246 18.9744 10.4271C18.9996 10.7356 19 11.1317 19 11.7V16.3C19 16.8683 18.9996 17.2645 18.9744 17.5729C18.9497 17.8755 18.9036 18.0493 18.8365 18.181C18.6927 18.4632 18.4632 18.6927 18.181 18.8365C18.0493 18.9036 17.8755 18.9497 17.5729 18.9744C17.2645 18.9996 16.8683 19 16.3 19H11.7C11.1317 19 10.7356 18.9996 10.4271 18.9744C10.1246 18.9497 9.95071 18.9036 9.81903 18.8365C9.53679 18.6927 9.30732 18.4632 9.16351 18.181C9.09642 18.0493 9.05033 17.8755 9.02561 17.5729C9.00041 17.2645 9.00002 16.8683 9.00002 16.3V11.7C9.00002 11.1317 9.00041 10.7356 9.02561 10.4271C9.05033 10.1246 9.09642 9.95071 9.16351 9.81903C9.30732 9.53679 9.53679 9.30732 9.81903 9.16351Z"
                    fill="white"
                    fill-opacity="0.2"
                  />
                </svg>
              </div>
              <div className="px-3 py-1 ml-4 text-xs font-bold text-black bg-white rounded-lg">
                Create key
              </div>
            </div>
          </div>
          <div className="pl-3 border-b-[0.75px] before:left-0 before:top-[30px] before:w-[85px] before:h-[1px] before:bg-white before:absolute relative border-white/15 mt-[34px] text-xs pb-[10px] flex flex-row gap-x-8">
            <p className="">Overview</p>
            <p className="text-white/40">Keys</p>
            <p className="text-white/40">Settings</p>
          </div>
          <div className="border-[0.75px] border-white/15 p-6 gap-x-6 mt-[32px] rounded-2xl flex">
            <div className="flex flex-col min-w-[100px]">
              <p className="text-xs text-white/40">Usage 30 days</p>
              <p className="pt-2 font-medium">7</p>
            </div>
            <div className="flex flex-col min-w-[100px]">
              <p className="text-xs text-white/40">Expires</p>
              <p className="pt-2 font-medium">-</p>
            </div>
            <div className="flex flex-col min-w-[100px]">
              <p className="text-xs text-white/40">Remaining</p>
              <p className="pt-2 font-medium">73</p>
            </div>
            <div className="flex flex-col min-w-[100px]">
              <p className="text-xs text-white/40">Last used</p>
              <p className="pt-2 font-medium">4d ago</p>
            </div>
            <div className="flex flex-col min-w-[100px]">
              <p className="text-xs text-white/40">Total uses</p>
              <p className="pt-2 font-medium">7</p>
            </div>
            <div className="flex flex-col">
              <p className="text-xs text-white/40">Key ID</p>
              <div className="flex items-center gap-2 px-2 py-1 font-mono text-xs rounded-md rounded-lg bg-white/5 font-sm text-white/40">
                api_UNWrXjYp6AF2H7Nx
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
                    d="M13 5.00002C13.4886 5.00002 13.6599 5.00244 13.7927 5.02884C14.3877 5.1472 14.8528 5.61235 14.9712 6.20738C14.9976 6.34011 15 6.5114 15 7.00002H16L16 6.94215V6.94213C16.0001 6.53333 16.0002 6.25469 15.952 6.01229C15.7547 5.02057 14.9795 4.24532 13.9877 4.04806C13.7453 3.99984 13.4667 3.99991 13.0579 4.00001L13 4.00002H7.70002H7.67861C7.13672 4.00001 6.69965 4.00001 6.34571 4.02893C5.98128 4.0587 5.66119 4.12161 5.36504 4.2725C4.89464 4.51219 4.51219 4.89464 4.2725 5.36504C4.12161 5.66119 4.0587 5.98128 4.02893 6.34571C4.00001 6.69965 4.00001 7.13672 4.00002 7.67862V7.70002V13L4.00001 13.0579C3.99991 13.4667 3.99984 13.7453 4.04806 13.9877C4.24532 14.9795 5.02057 15.7547 6.01229 15.952C6.25469 16.0002 6.53333 16.0001 6.94213 16H6.94215L7.00002 16V15C6.5114 15 6.34011 14.9976 6.20738 14.9712C5.61235 14.8528 5.1472 14.3877 5.02884 13.7927C5.00244 13.6599 5.00002 13.4886 5.00002 13V7.70002C5.00002 7.13172 5.00041 6.73556 5.02561 6.42714C5.05033 6.12455 5.09642 5.95071 5.16351 5.81903C5.30732 5.53679 5.53679 5.30732 5.81903 5.16351C5.95071 5.09642 6.12455 5.05033 6.42714 5.02561C6.73556 5.00041 7.13172 5.00002 7.70002 5.00002H13ZM11.7 8.00002H11.6786C11.1367 8.00001 10.6996 8.00001 10.3457 8.02893C9.98128 8.0587 9.66119 8.12161 9.36504 8.2725C8.89464 8.51219 8.51219 8.89464 8.2725 9.36504C8.12161 9.66119 8.0587 9.98128 8.02893 10.3457C8.00001 10.6996 8.00001 11.1367 8.00002 11.6786V11.7V16.3V16.3214C8.00001 16.8633 8.00001 17.3004 8.02893 17.6543C8.0587 18.0188 8.12161 18.3388 8.2725 18.635C8.51219 19.1054 8.89464 19.4879 9.36504 19.7275C9.66119 19.8784 9.98128 19.9413 10.3457 19.9711C10.6996 20 11.1366 20 11.6785 20H11.6786H11.7H16.3H16.3214H16.3216C16.8634 20 17.3004 20 17.6543 19.9711C18.0188 19.9413 18.3388 19.8784 18.635 19.7275C19.1054 19.4879 19.4879 19.1054 19.7275 18.635C19.8784 18.3388 19.9413 18.0188 19.9711 17.6543C20 17.3004 20 16.8634 20 16.3216V16.3214V16.3V11.7V11.6786V11.6785C20 11.1366 20 10.6996 19.9711 10.3457C19.9413 9.98128 19.8784 9.66119 19.7275 9.36504C19.4879 8.89464 19.1054 8.51219 18.635 8.2725C18.3388 8.12161 18.0188 8.0587 17.6543 8.02893C17.3004 8.00001 16.8633 8.00001 16.3214 8.00002H16.3H11.7ZM9.81903 9.16351C9.95071 9.09642 10.1246 9.05033 10.4271 9.02561C10.7356 9.00041 11.1317 9.00002 11.7 9.00002H16.3C16.8683 9.00002 17.2645 9.00041 17.5729 9.02561C17.8755 9.05033 18.0493 9.09642 18.181 9.16351C18.4632 9.30732 18.6927 9.53679 18.8365 9.81903C18.9036 9.95071 18.9497 10.1246 18.9744 10.4271C18.9996 10.7356 19 11.1317 19 11.7V16.3C19 16.8683 18.9996 17.2645 18.9744 17.5729C18.9497 17.8755 18.9036 18.0493 18.8365 18.181C18.6927 18.4632 18.4632 18.6927 18.181 18.8365C18.0493 18.9036 17.8755 18.9497 17.5729 18.9744C17.2645 18.9996 16.8683 19 16.3 19H11.7C11.1317 19 10.7356 18.9996 10.4271 18.9744C10.1246 18.9497 9.95071 18.9036 9.81903 18.8365C9.53679 18.6927 9.30732 18.4632 9.16351 18.181C9.09642 18.0493 9.05033 17.8755 9.02561 17.5729C9.00041 17.2645 9.00002 16.8683 9.00002 16.3V11.7C9.00002 11.1317 9.00041 10.7356 9.02561 10.4271C9.05033 10.1246 9.09642 9.95071 9.16351 9.81903C9.30732 9.53679 9.53679 9.30732 9.81903 9.16351Z"
                    fill="white"
                    fill-opacity="0.2"
                  />
                </svg>
              </div>
            </div>
          </div>
          <div className="border-[0.75px] border-white/15 p-6 gap-x-6 mt-[32px] rounded-2xl flex flex-col">
            <div className="flex items-center justify-between w-full text-xs text-white/30">
              <div className="min-w-[400px]">
                <h3 className="text-base font-medium text-white">Usage 30 days</h3>
                <p>See when this key was verified</p>
              </div>
              <div className="flex items-center">
                <div className="w-[6px] h-[6px] bg-white mr-3" />
                <p>Success</p>
              </div>
              <div className="flex items-center">
                <div className="w-[6px] h-[6px] bg-[#8F6424] mr-3" />
                <p>Rate limited</p>
              </div>
              <div className="flex items-center">
                <div className="w-[6px] h-[6px] bg-[#853A2D] mr-3" />
                <p>Usage exceeded</p>
              </div>
            </div>
            <div className="flex items-end mt-4 space-x-10 bar-chart">
              <div className="bg-white h-[20px] min-w-[10px] mb-[20px] ml-6" />
              <div className="bg-white h-[100px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[80px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[40px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[60px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[100px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[40px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[30px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[20px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[30px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[50px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[90px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[70px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[90px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[110px] min-w-[10px] mb-[20px]" />
              <div className="bg-white h-[100px] min-w-[10px] mb-[20px]" />
            </div>
          </div>
        </div>
      </div>
      {/* <div className="max-w-[1221px] relative overflow-hidden w-full analytics-background-gradient h-[600px] xl:h-[576px]  w-full  mr-10">
        <AnalyticsWebApp className="hidden md:flex" />
        <AnalyticsWebAppMobile />
      </div> */}
    </motion.div>
  );
}

export function Editor({
  codeBlock,
  language,
  theme,
}: { codeBlock: string; language: string; theme?: PrismTheme }) {
  return (
    <Highlight theme={theme} code={codeBlock} language={language}>
      {({ tokens, getLineProps, getTokenProps }) => (
        <pre className="leading-8">
          {tokens.map((line, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
            <div key={`${line}-${i}`} {...getLineProps({ line })}>
              <span className="line-number">{i + 1}</span>
              {line.map((token, key) => (
                <span key={`${key}-${token}`} {...getTokenProps({ token })} />
              ))}
            </div>
          ))}
        </pre>
      )}
    </Highlight>
  );
}

export function TerminalIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      className={className}
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M17.9864 5.36932C18.2076 5.13794 18.5073 5.00771 18.82 5.0072V5C19.1328 5.00051 19.4326 5.13074 19.6538 5.36214C19.875 5.59352 19.9993 5.90721 20 6.23444C20 6.82321 19.606 7.28113 19.0871 7.41197L13.5224 17.3725C13.5623 17.5034 13.5973 17.6342 13.5973 17.7651C13.5968 18.0923 13.4725 18.406 13.2513 18.6374C13.0301 18.8688 12.7303 18.999 12.4175 18.9995C12.1049 18.999 11.8048 18.8688 11.5839 18.6374C11.3627 18.406 11.2382 18.0923 11.2377 17.7651C11.2377 17.1704 11.6478 16.6974 12.183 16.5797L17.7195 6.65246C17.6752 6.52294 17.6402 6.38687 17.6402 6.24164C17.6409 5.9144 17.7652 5.60072 17.9864 5.36932ZM18.3643 6.71855C18.4852 6.84504 18.649 6.91611 18.82 6.91611C18.991 6.91611 19.155 6.84504 19.2758 6.71855C19.3967 6.59207 19.4646 6.42052 19.4646 6.24164C19.4646 6.06275 19.3967 5.89121 19.2758 5.76472C19.155 5.63824 18.991 5.56718 18.82 5.56718C18.649 5.56718 18.4852 5.63824 18.3643 5.76472C18.2435 5.89121 18.1754 6.06275 18.1754 6.24164C18.1754 6.42052 18.2435 6.59207 18.3643 6.71855ZM11.9616 18.242C12.0827 18.3685 12.2465 18.4396 12.4175 18.4396C12.5885 18.4396 12.7525 18.3685 12.8734 18.242C12.9942 18.1155 13.0621 17.9439 13.0621 17.7651C13.0621 17.5862 12.9942 17.4146 12.8734 17.2882C12.7525 17.1617 12.5885 17.0906 12.4175 17.0906C12.2465 17.0906 12.0827 17.1617 11.9616 17.2882C11.8408 17.4146 11.7729 17.5862 11.7729 17.7651C11.7729 17.9439 11.8408 18.1155 11.9616 18.242Z"
        fill="url(#paint0_linear_840_1992)"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M12.8178 6.24152C12.8178 5.56123 13.3435 5.00929 13.9937 5.00708C14.3044 5.00825 14.6024 5.13751 14.8231 5.36688C15.044 5.59687 15.169 5.90888 15.1714 6.23497C15.1714 6.82374 14.7773 7.28166 14.2585 7.4125L8.69378 17.3732C8.73367 17.5039 8.76867 17.6347 8.76867 17.7656C8.7682 18.0929 8.64385 18.4065 8.42268 18.6379C8.20151 18.8693 7.90172 18.9995 7.58887 19C7.27601 18.9995 6.97622 18.8693 6.75505 18.6377C6.53388 18.4061 6.4093 18.0923 6.40906 17.765C6.40906 17.1703 6.81921 16.6973 7.3544 16.5795L12.8972 6.65235C12.8528 6.52282 12.8178 6.38675 12.8178 6.24152ZM13.5418 6.71844C13.6629 6.84493 13.8266 6.91598 13.9976 6.91598C14.1686 6.91598 14.3327 6.84493 14.4535 6.71844C14.5744 6.59195 14.6422 6.4204 14.6422 6.24152C14.6422 6.06264 14.5744 5.89108 14.4535 5.7646C14.3327 5.63811 14.1686 5.56705 13.9976 5.56705C13.8266 5.56705 13.6629 5.63811 13.5418 5.7646C13.4209 5.89108 13.353 6.06264 13.353 6.24152C13.353 6.4204 13.4209 6.59195 13.5418 6.71844ZM7.133 18.2425C7.25408 18.369 7.41786 18.4401 7.58887 18.4401C7.75988 18.4401 7.92389 18.369 8.04474 18.2425C8.16559 18.116 8.23348 17.9446 8.23348 17.7656C8.23348 17.5867 8.16559 17.4151 8.04474 17.2887C7.92389 17.1622 7.75988 17.0912 7.58887 17.0912C7.41786 17.0912 7.25408 17.1622 7.133 17.2887C7.01215 17.4151 6.94426 17.5867 6.94426 17.7656C6.94426 17.9446 7.01215 18.116 7.133 18.2425Z"
        fill="url(#paint1_linear_840_1992)"
      />
      <path
        d="M13.9937 5.00708C13.9951 5.00708 13.9962 5.00707 13.9976 5.00707H13.9913C13.992 5.00707 13.993 5.00708 13.9937 5.00708Z"
        fill="url(#paint2_linear_840_1992)"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M6.00732 10.5274C5.78615 10.7588 5.48636 10.889 5.1735 10.8895L5.1798 10.883C4.86718 10.8824 4.56715 10.7522 4.34622 10.5208C4.12505 10.2894 4.00047 9.97574 4 9.64851C4 9.0532 4.40384 8.58088 4.93787 8.46312C4.96377 8.45755 4.98896 8.45029 5.01416 8.44305C5.06572 8.42819 5.11704 8.41341 5.17304 8.41341C5.69376 8.41341 6.1221 8.7719 6.27842 9.26253C6.31784 9.39337 6.35354 9.52422 6.35354 9.65506C6.35284 9.98228 6.22849 10.296 6.00732 10.5274ZM5.62937 9.17814C5.50852 9.05165 5.34451 8.98059 5.1735 8.98059C5.00273 8.98059 4.83872 9.05165 4.71787 9.17814C4.59678 9.30461 4.52889 9.47617 4.52889 9.65506C4.52889 9.83392 4.59678 10.0055 4.71787 10.132C4.83872 10.2585 5.00273 10.3295 5.1735 10.3295C5.34451 10.3295 5.50852 10.2585 5.62937 10.132C5.75022 10.0055 5.81811 9.83392 5.81811 9.65506C5.81811 9.47617 5.75022 9.30461 5.62937 9.17814Z"
        fill="url(#paint3_linear_840_1992)"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M6.27842 13.5769C6.31784 13.7077 6.35354 13.8386 6.35354 13.9694L6.3596 13.9628C6.35914 14.2901 6.23479 14.6038 6.01362 14.8352C5.79245 15.0666 5.49266 15.1968 5.1798 15.1973C4.86718 15.1968 4.56715 15.0666 4.34622 14.8352C4.12505 14.6038 4.00047 14.2901 4 13.9628C4 13.3675 4.40384 12.8946 4.93787 12.7768C4.96423 12.7713 4.99013 12.764 5.01579 12.7567C5.06688 12.7422 5.11774 12.7277 5.17304 12.7277C5.69446 12.7277 6.1221 13.0862 6.27842 13.5769ZM4.71787 13.4925C4.59678 13.619 4.52889 13.7905 4.52889 13.9694C4.52889 14.1483 4.59678 14.3198 4.71787 14.4463C4.83872 14.5728 5.00273 14.6439 5.1735 14.6439C5.34451 14.6439 5.50852 14.5728 5.62937 14.4463C5.75022 14.3198 5.81811 14.1483 5.81811 13.9694C5.81811 13.7905 5.75022 13.619 5.62937 13.4925C5.50852 13.366 5.34451 13.2949 5.1735 13.2949C5.00273 13.2949 4.83872 13.366 4.71787 13.4925Z"
        fill="url(#paint4_linear_840_1992)"
      />
      <defs>
        <linearGradient
          id="paint0_linear_840_1992"
          x1="4.15606"
          y1="3.49029"
          x2="4.15606"
          y2="19.8307"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.4" />
          <stop offset="1" stopColor="white" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_840_1992"
          x1="4.15606"
          y1="3.49029"
          x2="4.15606"
          y2="19.8307"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.4" />
          <stop offset="1" stopColor="white" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_840_1992"
          x1="4.15606"
          y1="3.49029"
          x2="4.15606"
          y2="19.8307"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.4" />
          <stop offset="1" stopColor="white" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_840_1992"
          x1="4.15606"
          y1="3.49029"
          x2="4.15606"
          y2="19.8307"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.4" />
          <stop offset="1" stopColor="white" />
        </linearGradient>
        <linearGradient
          id="paint4_linear_840_1992"
          x1="4.15606"
          y1="3.49029"
          x2="4.15606"
          y2="19.8307"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.4" />
          <stop offset="1" stopColor="white" />
        </linearGradient>
      </defs>
    </svg>
  );
}

export function BentoText() {
  return (
    <div className="flex flex-col text-white absolute left-[20px] sm:left-[40px] xl:left-[40px] bottom-[40px] max-w-[286px]">
      <div className="flex items-center w-full">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          className="w-6 h-6"
        >
          <path
            fillRule="evenodd"
            clipRule="evenodd"
            d="M23 5.5C23 6.88071 21.8807 8 20.5 8C19.1193 8 18 6.88071 18 5.5C18 4.11929 19.1193 3 20.5 3C21.8807 3 23 4.11929 23 5.5ZM22 5.5C22 6.32843 21.3284 7 20.5 7C19.6716 7 19 6.32843 19 5.5C19 4.67157 19.6716 4 20.5 4C21.3284 4 22 4.67157 22 5.5ZM6.6786 4L6.7 4H15V5H6.7C6.1317 5 5.73554 5.00039 5.42712 5.02559C5.12454 5.05031 4.95069 5.0964 4.81901 5.16349C4.53677 5.3073 4.3073 5.53677 4.16349 5.81901C4.0964 5.95069 4.05031 6.12454 4.02559 6.42712C4.00039 6.73554 4 7.1317 4 7.7V16.3C4 16.8683 4.00039 17.2645 4.02559 17.5729C4.05031 17.8755 4.0964 18.0493 4.16349 18.181C4.3073 18.4632 4.53677 18.6927 4.81901 18.8365C4.95069 18.9036 5.12454 18.9497 5.42712 18.9744C5.73554 18.9996 6.1317 19 6.7 19H18.3C18.8683 19 19.2645 18.9996 19.5729 18.9744C19.8755 18.9497 20.0493 18.9036 20.181 18.8365C20.4632 18.6927 20.6927 18.4632 20.8365 18.181C20.9036 18.0493 20.9497 17.8755 20.9744 17.5729C20.9996 17.2645 21 16.8683 21 16.3V11H22V16.3V16.3214C22 16.8633 22 17.3004 21.9711 17.6543C21.9413 18.0187 21.8784 18.3388 21.7275 18.635C21.4878 19.1054 21.1054 19.4878 20.635 19.7275C20.3388 19.8784 20.0187 19.9413 19.6543 19.9711C19.3004 20 18.8635 20 18.3217 20H18.3216H18.3216H18.3216H18.3214H18.3H6.7H6.67858H6.67844H6.67839H6.67835H6.67831C6.13655 20 5.69957 20 5.34569 19.9711C4.98126 19.9413 4.66117 19.8784 4.36502 19.7275C3.89462 19.4878 3.51217 19.1054 3.27248 18.635C3.12159 18.3388 3.05868 18.0187 3.02891 17.6543C2.99999 17.3004 3 16.8633 3 16.3214V16.3V7.7V7.6786C3 7.1367 2.99999 6.69963 3.02891 6.34569C3.05868 5.98126 3.12159 5.66117 3.27248 5.36502C3.51217 4.89462 3.89462 4.51217 4.36502 4.27248C4.66117 4.12159 4.98126 4.05868 5.34569 4.02891C5.69963 3.99999 6.1367 3.99999 6.6786 4ZM8 16V8H9V16H8ZM12 12V16H13V12H12ZM16 16V10H17V16H16Z"
            fill="white"
            fillOpacity="0.4"
          />
        </svg>
        <h3 className="ml-4 text-lg font-medium text-white">Realtime Analytics</h3>
      </div>
      <p className="mt-4 leading-6 text-white/60">
        Access real-time insights into your API usage through our dashboard, or build your own on
        top of our APi.
      </p>
    </div>
  );
}

export function LightSvg({ className }: { className?: string }) {
  return (
    <svg
      width={706}
      height={773}
      viewBox="0 0 706 773"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      <g opacity={0.2}>
        <g
          style={{
            mixBlendMode: "lighten",
          }}
          filter="url(#a)"
        >
          <ellipse
            cx={24.471}
            cy={219.174}
            rx={24.471}
            ry={219.174}
            transform="rotate(15.067 -346.215 1160.893)skewX(.027)"
            fill="url(#b)"
            fillOpacity={0.5}
          />
        </g>
        <g
          style={{
            mixBlendMode: "color-dodge",
          }}
          filter="url(#c)"
        >
          <ellipse
            cx={355.612}
            cy={332.214}
            rx={19.782}
            ry={219.116}
            fill="url(#d)"
            fillOpacity={0.5}
          />
        </g>
        <g
          style={{
            mixBlendMode: "lighten",
          }}
          filter="url(#e)"
        >
          <ellipse
            cx={16.707}
            cy={284.877}
            rx={16.707}
            ry={284.877}
            transform="rotate(-15.013 621.533 -1357.818)skewX(-.027)"
            fill="url(#f)"
            fillOpacity={0.5}
          />
        </g>
        <g
          style={{
            mixBlendMode: "lighten",
          }}
          filter="url(#g)"
        >
          <ellipse
            cx={16.707}
            cy={134.986}
            rx={16.707}
            ry={134.986}
            transform="rotate(-15.013 606.533 -1243.985)skewX(-.027)"
            fill="url(#h)"
            fillOpacity={0.5}
          />
        </g>
        <g
          style={{
            mixBlendMode: "lighten",
          }}
          filter="url(#i)"
        >
          <ellipse
            cx={353.187}
            cy={420.944}
            rx={16.61}
            ry={285.056}
            fill="url(#j)"
            fillOpacity={0.5}
          />
        </g>
        <g
          style={{
            mixBlendMode: "lighten",
          }}
          filter="url(#k)"
        >
          <ellipse
            cx={353.187}
            cy={420.944}
            rx={16.61}
            ry={285.056}
            fill="url(#l)"
            fillOpacity={0.5}
          />
        </g>
        <g
          style={{
            mixBlendMode: "lighten",
          }}
          filter="url(#m)"
        >
          <ellipse cx={353} cy={253.199} rx={240} ry={140.1} fill="url(#n)" fillOpacity={0.5} />
        </g>
        <g
          style={{
            mixBlendMode: "lighten",
          }}
          filter="url(#o)"
        >
          <ellipse
            cx={353}
            cy={184.456}
            rx={119.813}
            ry={71.357}
            fill="url(#p)"
            fillOpacity={0.5}
          />
        </g>
        <g
          style={{
            mixBlendMode: "lighten",
          }}
          filter="url(#q)"
        >
          <ellipse
            cx={353}
            cy={189.873}
            rx={100.778}
            ry={59.963}
            fill="url(#r)"
            fillOpacity={0.5}
          />
        </g>
      </g>
      <defs>
        <linearGradient
          id="b"
          x1={24.471}
          y1={0}
          x2={24.471}
          y2={438.347}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <linearGradient
          id="d"
          x1={355.612}
          y1={113.099}
          x2={355.612}
          y2={551.33}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <linearGradient
          id="f"
          x1={16.707}
          y1={0}
          x2={16.707}
          y2={569.753}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <linearGradient
          id="h"
          x1={16.707}
          y1={0}
          x2={16.707}
          y2={269.972}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <linearGradient
          id="j"
          x1={353.187}
          y1={135.888}
          x2={353.187}
          y2={706}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <linearGradient
          id="l"
          x1={353.187}
          y1={135.888}
          x2={353.187}
          y2={706}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <linearGradient
          id="n"
          x1={353}
          y1={113.099}
          x2={353}
          y2={393.298}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <linearGradient
          id="p"
          x1={353}
          y1={113.099}
          x2={353}
          y2={255.814}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <linearGradient
          id="r"
          x1={353}
          y1={129.91}
          x2={353}
          y2={249.836}
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset={1} stopColor="#fff" stopOpacity={0} />
        </linearGradient>
        <filter
          id="a"
          x={128.273}
          y={69.424}
          width={256.71}
          height={557.026}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={33.375} result="effect1_foregroundBlur_840_2403" />
        </filter>
        <filter
          id="c"
          x={269.08}
          y={46.349}
          width={173.065}
          height={571.731}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={33.375} result="effect1_foregroundBlur_840_2403" />
        </filter>
        <filter
          id="e"
          x={320.576}
          y={43.543}
          width={284.36}
          height={683.943}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={33.375} result="effect1_foregroundBlur_840_2403" />
        </filter>
        <filter
          id="g"
          x={288.78}
          y={43.505}
          width={210.431}
          height={394.435}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={33.375} result="effect1_foregroundBlur_840_2403" />
        </filter>
        <filter
          id="i"
          x={269.827}
          y={69.138}
          width={166.719}
          height={703.612}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={33.375} result="effect1_foregroundBlur_840_2403" />
        </filter>
        <filter
          id="k"
          x={269.827}
          y={69.138}
          width={166.719}
          height={703.612}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={33.375} result="effect1_foregroundBlur_840_2403" />
        </filter>
        <filter
          id="m"
          x={0.5}
          y={0.599}
          width={705}
          height={505.199}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={56.25} result="effect1_foregroundBlur_840_2403" />
        </filter>
        <filter
          id="o"
          x={120.687}
          y={0.599}
          width={464.627}
          height={367.715}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={56.25} result="effect1_foregroundBlur_840_2403" />
        </filter>
        <filter
          id="q"
          x={139.723}
          y={17.41}
          width={426.555}
          height={344.925}
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity={0} result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation={56.25} result="effect1_foregroundBlur_840_2403" />
        </filter>
      </defs>
    </svg>
  );
}
