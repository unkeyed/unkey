export default function SVGAccelerateMini(props: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="88"
      height="79"
      fill="none"
      viewBox="0 0 88 79"
      {...props}
    >
      <circle
        cx="56"
        cy="47"
        r="24"
        stroke="#FF1D4F"
        strokeDasharray="2 3"
        strokeOpacity="0.15"
        strokeWidth="16"
      />
      <g filter="url(#filter0_f_5085_41)" opacity="0.5">
        <path fill="url(#paint0_angular_5085_41)" d="M24 47a32 32 0 019.373-22.627L56 47H24z" />
      </g>
      <mask
        id="mask0_5085_41"
        style={{ maskType: "alpha" }}
        width="32"
        height="23"
        x="24"
        y="24"
        maskUnits="userSpaceOnUse"
      >
        <path fill="#D9D9D9" d="M24 47a32 32 0 019.373-22.627L56 47H24z" />
      </mask>
      <g mask="url(#mask0_5085_41)">
        <circle
          cx="56"
          cy="47"
          r="24"
          stroke="url(#paint1_angular_5085_41)"
          strokeDasharray="2 3"
          strokeWidth="16"
        />
      </g>
      <path stroke="url(#paint2_linear_5085_41)" strokeWidth="2" d="M33.371 24.373l45.255 45.254" />
      <defs>
        <filter
          id="filter0_f_5085_41"
          width="80"
          height="70.627"
          x="0"
          y="0.373"
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur result="effect1_foregroundBlur_5085_41" stdDeviation="12" />
        </filter>
        <radialGradient
          id="paint0_angular_5085_41"
          cx="0"
          cy="0"
          r="1"
          gradientTransform="matrix(22.5 22.5 -22.5 22.5 56 47)"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stopColor="#FF1D4F" />
          <stop offset="1" stopColor="#7002FC" />
        </radialGradient>
        <radialGradient
          id="paint1_angular_5085_41"
          cx="0"
          cy="0"
          r="1"
          gradientTransform="rotate(43.546 -30.834 93.6) scale(30.8026)"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#7002FC" />
          <stop offset="0" stopColor="#FF1D4F" stopOpacity="0" />
          <stop offset="0" stopColor="#FF1D4F" stopOpacity="0.1" />
          <stop offset="0.352" stopColor="#FF1D4F" stopOpacity="0.1" />
          <stop offset="0.378" stopColor="#FF1D4F" stopOpacity="0.1" />
          <stop offset="0.381" stopColor="#FF1D4F" />
          <stop offset="1" stopColor="#7002FC" />
          <stop offset="1" stopColor="#7002FC" stopOpacity="0" />
        </radialGradient>
        <linearGradient
          id="paint2_linear_5085_41"
          x1="33.725"
          x2="78.98"
          y1="24.019"
          y2="69.274"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#fff" />
          <stop offset="0.5" stopColor="#fff" stopOpacity="0" />
          <stop offset="0.5" stopColor="#fff" stopOpacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
}
