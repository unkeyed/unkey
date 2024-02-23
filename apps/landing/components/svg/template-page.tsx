import { cn } from "@/lib/utils";
import { PropsWithChildren } from "react";

export const SparkleIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    width="17"
    height="16"
    viewBox="0 0 17 16"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g clip-path="url(#clip0_4006_10078)">
      <path
        d="M13.5 0.75C13.5 1.89705 12.3971 3 11.25 3C12.3971 3 13.5 4.10295 13.5 5.25C13.5 4.10295 14.6029 3 15.75 3C14.6029 3 13.5 1.89705 13.5 0.75Z"
        stroke="white"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        d="M13.5 10.75C13.5 11.8971 12.3971 13 11.25 13C12.3971 13 13.5 14.1029 13.5 15.25C13.5 14.1029 14.6029 13 15.75 13C14.6029 13 13.5 11.8971 13.5 10.75Z"
        stroke="white"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        d="M5.5 3.75C5.5 5.91666 3.41666 8 1.25 8C3.41666 8 5.5 10.0833 5.5 12.25C5.5 10.0833 7.5833 8 9.75 8C7.5833 8 5.5 5.91666 5.5 3.75Z"
        stroke="white"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </g>
    <defs>
      <clipPath id="clip0_4006_10078">
        <rect width="16" height="16" fill="white" transform="translate(0.5)" />
      </clipPath>
    </defs>
  </svg>
);

export const TemplatesRightArrow = ({ className }: { className?: string }) => (
  <svg
    className={className}
    width="24"
    height="24"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path
      d="M14.5 8.5L18.5 12.5M18.5 12.5L14.5 16.5M18.5 12.5H6.5"
      stroke="white"
      stroke-opacity="0.4"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
);

export const CodeIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    width="24"
    height="24"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <rect width="24" height="24" rx="6" fill="white" fill-opacity="0.1" />
    <path
      d="M15.75 8.75L19.25 12L15.75 15.25M8.25 8.75L4.75 12L8.25 15.25M13.25 5.75L10.75 18.25"
      stroke="white"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
);

export const FrameworkIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    width="24"
    height="24"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <rect width="24" height="24" rx="6" fill="white" fill-opacity="0.1" />
    <path
      d="M8 8H8.01M8 16H8.01M16 8H16.01M16 16H16.01M13.25 12C13.25 12.3315 13.1183 12.6495 12.8839 12.8839C12.6495 13.1183 12.3315 13.25 12 13.25C11.6685 13.25 11.3505 13.1183 11.1161 12.8839C10.8817 12.6495 10.75 12.3315 10.75 12C10.75 11.6685 10.8817 11.3505 11.1161 11.1161C11.3505 10.8817 11.6685 10.75 12 10.75C12.3315 10.75 12.6495 10.8817 12.8839 11.1161C13.1183 11.3505 13.25 11.6685 13.25 12ZM13.25 6C13.25 6.33152 13.1183 6.64946 12.8839 6.88388C12.6495 7.1183 12.3315 7.25 12 7.25C11.6685 7.25 11.3505 7.1183 11.1161 6.88388C10.8817 6.64946 10.75 6.33152 10.75 6C10.75 5.66848 10.8817 5.35054 11.1161 5.11612C11.3505 4.8817 11.6685 4.75 12 4.75C12.3315 4.75 12.6495 4.8817 12.8839 5.11612C13.1183 5.35054 13.25 5.66848 13.25 6ZM19.25 12C19.25 12.3315 19.1183 12.6495 18.8839 12.8839C18.6495 13.1183 18.3315 13.25 18 13.25C17.6685 13.25 17.3505 13.1183 17.1161 12.8839C16.8817 12.6495 16.75 12.3315 16.75 12C16.75 11.6685 16.8817 11.3505 17.1161 11.1161C17.3505 10.8817 17.6685 10.75 18 10.75C18.3315 10.75 18.6495 10.8817 18.8839 11.1161C19.1183 11.3505 19.25 11.6685 19.25 12ZM7.25 12C7.25 12.3315 7.1183 12.6495 6.88388 12.8839C6.64946 13.1183 6.33152 13.25 6 13.25C5.66848 13.25 5.35054 13.1183 5.11612 12.8839C4.8817 12.6495 4.75 12.3315 4.75 12C4.75 11.6685 4.8817 11.3505 5.11612 11.1161C5.35054 10.8817 5.66848 10.75 6 10.75C6.33152 10.75 6.64946 10.8817 6.88388 11.1161C7.1183 11.3505 7.25 11.6685 7.25 12ZM13.25 18C13.25 18.3315 13.1183 18.6495 12.8839 18.8839C12.6495 19.1183 12.3315 19.25 12 19.25C11.6685 19.25 11.3505 19.1183 11.1161 18.8839C10.8817 18.6495 10.75 18.3315 10.75 18C10.75 17.6685 10.8817 17.3505 11.1161 17.1161C11.3505 16.8817 11.6685 16.75 12 16.75C12.3315 16.75 12.6495 16.8817 12.8839 17.1161C13.1183 17.3505 13.25 17.6685 13.25 18Z"
      stroke="white"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
);

export const TemplateTopLight = ({
  className,
}: {
  className?: string;
}) => (
  <svg
    className={cn("absolute top-0 left-0 right-0 pointer-events-none overflow-hidden", className)}
    viewBox="0 0 942 622"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter0_f_4046_1775)">
      <ellipse
        cx="369.193"
        cy="53.7926"
        rx="32.7783"
        ry="293.346"
        transform="rotate(15.0538 369.193 53.7926)"
        fill="url(#paint0_linear_4046_1775)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter1_f_4046_1775)">
      <ellipse
        cx="475"
        cy="32.7505"
        rx="26.5"
        ry="293.25"
        fill="url(#paint1_linear_4046_1775)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter2_f_4046_1775)">
      <ellipse
        cx="578.34"
        cy="104.085"
        rx="22.3794"
        ry="381.284"
        transform="rotate(-15 578.34 104.085)"
        fill="url(#paint2_linear_4046_1775)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter3_f_4046_1775)">
      <ellipse
        cx="526.416"
        cy="-89.6963"
        rx="22.3794"
        ry="180.667"
        transform="rotate(-15 526.416 -89.6963)"
        fill="url(#paint3_linear_4046_1775)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter4_f_4046_1775)">
      <ellipse
        cx="471.75"
        cy="151.501"
        rx="22.25"
        ry="381.5"
        fill="url(#paint4_linear_4046_1775)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter5_f_4046_1775)">
      <ellipse
        cx="471.5"
        cy="-72.9993"
        rx="321.5"
        ry="187.5"
        fill="url(#paint5_linear_4046_1775)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter6_f_4046_1775)">
      <ellipse
        cx="471.5"
        cy="-164.999"
        rx="160.5"
        ry="95.5"
        fill="url(#paint6_linear_4046_1775)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter7_f_4046_1775)">
      <ellipse
        cx="471.5"
        cy="-157.749"
        rx="135"
        ry="80.25"
        fill="url(#paint7_linear_4046_1775)"
        fill-opacity="0.5"
      />
    </g>
    <defs>
      <filter
        id="filter0_f_4046_1775"
        x="197.668"
        y="-318.616"
        width="343.05"
        height="744.818"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_4046_1775" />
      </filter>
      <filter
        id="filter1_f_4046_1775"
        x="359.5"
        y="-349.499"
        width="231"
        height="764.5"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_4046_1775" />
      </filter>
      <filter
        id="filter2_f_4046_1775"
        x="388.295"
        y="-353.254"
        width="380.09"
        height="914.676"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_4046_1775" />
      </filter>
      <filter
        id="filter3_f_4046_1775"
        x="385.889"
        y="-353.306"
        width="281.055"
        height="527.218"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_4046_1775" />
      </filter>
      <filter
        id="filter4_f_4046_1775"
        x="360.5"
        y="-318.999"
        width="222.5"
        height="941"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_4046_1775" />
      </filter>
      <filter
        id="filter5_f_4046_1775"
        x="-0.00012207"
        y="-410.499"
        width="943"
        height="675"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="75" result="effect1_foregroundBlur_4046_1775" />
      </filter>
      <filter
        id="filter6_f_4046_1775"
        x="161"
        y="-410.499"
        width="621"
        height="491"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="75" result="effect1_foregroundBlur_4046_1775" />
      </filter>
      <filter
        id="filter7_f_4046_1775"
        x="186.5"
        y="-387.999"
        width="570"
        height="460.5"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="75" result="effect1_foregroundBlur_4046_1775" />
      </filter>
      <linearGradient
        id="paint0_linear_4046_1775"
        x1="369.193"
        y1="-239.553"
        x2="369.193"
        y2="347.138"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint1_linear_4046_1775"
        x1="475"
        y1="-260.499"
        x2="475"
        y2="326.001"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint2_linear_4046_1775"
        x1="578.34"
        y1="-277.199"
        x2="578.34"
        y2="485.368"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint3_linear_4046_1775"
        x1="526.416"
        y1="-270.364"
        x2="526.416"
        y2="90.9709"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint4_linear_4046_1775"
        x1="471.75"
        y1="-229.999"
        x2="471.75"
        y2="533.001"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint5_linear_4046_1775"
        x1="471.5"
        y1="-260.499"
        x2="471.5"
        y2="114.501"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint6_linear_4046_1775"
        x1="471.5"
        y1="-260.499"
        x2="471.5"
        y2="-69.4993"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint7_linear_4046_1775"
        x1="471.5"
        y1="-237.999"
        x2="471.5"
        y2="-77.4994"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
    </defs>
  </svg>
);
