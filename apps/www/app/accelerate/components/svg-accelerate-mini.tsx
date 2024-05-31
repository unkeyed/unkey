export default function SVGAccelerateMini(props: { className?: string }) {
  return (
    <svg
      width="112"
      height="103"
      viewBox="0 0 112 103"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <circle
        cx="56"
        cy="56"
        r="24"
        stroke="#FF1D4F"
        stroke-opacity="0.15"
        stroke-width="16"
        stroke-dasharray="2 3"
      />
      <g opacity="0.5" filter="url(#filter0_f_4975_434)">
        <path
          d="M24 56C24 50.7376 25.2978 45.5564 27.7785 40.9153C30.2592 36.2743 33.8462 32.3166 38.2218 29.393C42.5973 26.4693 47.6264 24.6699 52.8635 24.1541C58.1005 23.6383 63.384 24.422 68.2459 26.4359C73.1077 28.4497 77.3979 31.6315 80.7363 35.6994C84.0748 39.7673 86.3585 44.5958 87.3851 49.7571C88.4118 54.9184 88.1497 60.2533 86.6221 65.2891C85.0945 70.3249 82.3485 74.9063 78.6274 78.6274L56 56L24 56Z"
          fill="url(#paint0_angular_4975_434)"
        />
      </g>
      <mask
        id="mask0_4975_434"
        style={{ maskType: "alpha" }}
        maskUnits="userSpaceOnUse"
        x="24"
        y="24"
        width="64"
        height="55"
      >
        <path
          d="M24 56C24 50.7376 25.2978 45.5564 27.7785 40.9153C30.2592 36.2743 33.8462 32.3166 38.2218 29.393C42.5973 26.4693 47.6264 24.6699 52.8635 24.1541C58.1005 23.6383 63.384 24.422 68.2459 26.4359C73.1077 28.4497 77.3979 31.6315 80.7363 35.6994C84.0748 39.7673 86.3585 44.5958 87.3851 49.7571C88.4118 54.9184 88.1497 60.2533 86.6221 65.2891C85.0945 70.3249 82.3485 74.9063 78.6274 78.6274L56 56L24 56Z"
          fill="#D9D9D9"
        />
      </mask>
      <g mask="url(#mask0_4975_434)">
        <circle
          cx="56"
          cy="56"
          r="24"
          stroke="url(#paint1_angular_4975_434)"
          stroke-width="16"
          stroke-dasharray="2 3"
        />
      </g>
      <path
        d="M78.627 78.6274L33.3721 33.3726"
        stroke="url(#paint2_linear_4975_434)"
        stroke-width="2"
      />
      <defs>
        <filter
          id="filter0_f_4975_434"
          x="0"
          y="0"
          width="112"
          height="102.627"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="12" result="effect1_foregroundBlur_4975_434" />
        </filter>
        <radialGradient
          id="paint0_angular_4975_434"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(56 56) rotate(45) scale(31.8198)"
        >
          <stop offset="0.0001" stop-color="#FF1D4F" />
          <stop offset="0.9997" stop-color="#7002FC" />
        </radialGradient>
        <radialGradient
          id="paint1_angular_4975_434"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(56 56) rotate(43.5461) scale(30.8026)"
        >
          <stop stop-color="#7002FC" />
          <stop offset="0.0001" stop-color="#FF1D4F" stop-opacity="0" />
          <stop offset="0.0002" stop-color="#FF1D4F" stop-opacity="0.1" />
          <stop offset="0.352076" stop-color="#FF1D4F" stop-opacity="0.1" />
          <stop offset="0.378012" stop-color="#FF1D4F" stop-opacity="0.1" />
          <stop offset="0.381003" stop-color="#FF1D4F" />
          <stop offset="0.9997" stop-color="#7002FC" />
          <stop offset="0.9998" stop-color="#7002FC" stop-opacity="0" />
        </radialGradient>
        <linearGradient
          id="paint2_linear_4975_434"
          x1="78.2734"
          y1="78.981"
          x2="33.0186"
          y2="33.7262"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="0.5" stop-color="white" stop-opacity="0" />
          <stop offset="0.5" stop-color="white" stop-opacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
}
