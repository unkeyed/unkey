export function RateLimitsBento() {
  return (
    <div className="w-full md:mt-5 relative border-[.75px] h-[520px] rounded-[32px] border-[#ffffff]/10 flex overflow-x-hidden rate-limits-background-gradient bg-gradient-to-t backdrop-blur-[1px] from-black/20 via-black/20 via-20% to-transparent">
      <RateLimits />
      <RateLimitsText />
    </div>
  );
}

export function RateLimits() {
  return (
    <div className="mx-[40px] flex w-full flex-col">
      <div className="flex h-[200px] w-full ratelimits-editor-bg-gradient rounded-b-xl ">
        <div className="flex flex-col font-mono text-sm text-white px-[24px] space-y-3 mt-1 border-r-[.75px] border-[#ffffff]/20">
          <p>1</p>
          <p>2</p>
          <p>3</p>
          <p>4</p>
          <p>5</p>
          <p>6</p>
        </div>
        <div className="flex w-full pl-8 font-mono text-xs leading-8 text-white whitespace-pre ratelimits-editor-bg-gradient-2 rounded-br-xl">
          {JSON.stringify({ rateLimit: { limit: 10, interval: 1000 } }, null, 2)}
        </div>
      </div>
      <div className="flex flex-col mt-8 ratelimits-fade-gradient">
        <div className="flex items-center">
          <div className="font-mono text-xs text-white sm:font-sm whitespace-nowrap">
            <span className="text-[#ffffff]/40">Rate limiting</span>
            <span className="text-[#ffffff]/40 tracking-[-5px]">...</span>
            <span className="inline-flex w-[4px] h-[12px] bg-white ratelimits-bar-shadow ml-3 relative top-[1px] flicker" />
          </div>
          <div className="inline-flex items-center overflow-hidden ml-4 h-[36px] text-white font-mono text-sm ratelimits-key-gradient border-[.75px] border-[#ffffff]/20 rounded-xl">
            <div className="w-[62px] h-[36px]">
              <svg
                className="ratelimits-key-icon "
                xmlns="http://www.w3.org/2000/svg"
                width="62"
                height="36"
                viewBox="0 0 62 36"
                fill="none"
              >
                <g filter="url(#filter0_d_840_1930)">
                  <rect
                    x="8"
                    y="6"
                    width="24"
                    height="24"
                    rx="6"
                    fill="#3CEEAE"
                    shapeRendering="crispEdges"
                  />
                  <rect
                    x="8"
                    y="6"
                    width="24"
                    height="24"
                    rx="6"
                    fill="black"
                    fillOpacity="0.15"
                    shapeRendering="crispEdges"
                  />
                  <rect
                    x="8.375"
                    y="6.375"
                    width="23.25"
                    height="23.25"
                    rx="5.625"
                    stroke="white"
                    strokeOpacity="0.1"
                    strokeWidth="0.75"
                    shapeRendering="crispEdges"
                  />
                  <path
                    d="M21.5 15L23 16.5M14.5 23.5H17.5V21.5H19.5V20.5L21.5 18.5L19.5 16.5L14.5 21.5V23.5ZM18 15L23 20L26.5 16.5L21.5 11.5L18 15Z"
                    stroke="white"
                  />
                </g>
                <defs>
                  <filter
                    id="filter0_d_840_1930"
                    x="-22"
                    y="-24"
                    width="84"
                    height="84"
                    filterUnits="userSpaceOnUse"
                    colorInterpolationFilters="sRGB"
                  >
                    <feFlood floodOpacity="0" result="BackgroundImageFix" />
                    <feColorMatrix
                      in="SourceAlpha"
                      type="matrix"
                      values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                      result="hardAlpha"
                    />
                    <feOffset />
                    <feGaussianBlur stdDeviation="15" />
                    <feComposite in2="hardAlpha" operator="out" />
                    <feColorMatrix
                      type="matrix"
                      values="0 0 0 0 0.235294 0 0 0 0 0.933333 0 0 0 0 0.682353 0 0 0 1 0"
                    />
                    <feBlend
                      mode="normal"
                      in2="BackgroundImageFix"
                      result="effect1_dropShadow_840_1930"
                    />
                    <feBlend
                      mode="normal"
                      in="SourceGraphic"
                      in2="effect1_dropShadow_840_1930"
                      result="shape"
                    />
                  </filter>
                </defs>
              </svg>
            </div>
            <p className="relative text-xs right-4">sk_TEwCE9AY9BFTq1XJdIO</p>
          </div>
        </div>
        <div className="inline-flex w-[205px] opacity-70 items-center ml-[150px] mt-[30px] h-[36px] text-white font-mono text-sm ratelimits-link ratelimits-key-gradient border-[.75px] border-[#ffffff]/20 rounded-xl [mask-image:linear-gradient(to_bottom_left,black_30%,transparent)]">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="84"
            height="84"
            viewBox="0 0 84 84"
            fill="none"
          >
            <g filter="url(#filter0_d_61_98)">
              <rect
                x="30"
                y="30"
                width="24"
                height="24"
                rx="6"
                fill="#6E56CF"
                shape-rendering="crispEdges"
              />
              <rect
                x="30"
                y="30"
                width="24"
                height="24"
                rx="6"
                fill="black"
                fill-opacity="0.15"
                shape-rendering="crispEdges"
              />
              <rect
                x="30.375"
                y="30.375"
                width="23.25"
                height="23.25"
                rx="5.625"
                stroke="white"
                stroke-opacity="0.1"
                stroke-width="0.75"
                shape-rendering="crispEdges"
              />
              <path
                d="M46.9 38H38.1C37.4925 38 37 38.4477 37 39V45C37 45.5523 37.4925 46 38.1 46H46.9C47.5075 46 48 45.5523 48 45V39C48 38.4477 47.5075 38 46.9 38Z"
                stroke="white"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
              <path
                d="M48 40L43.0665 42.8519C42.8967 42.9487 42.7004 43 42.5 43C42.2996 43 42.1033 42.9487 41.9335 42.8519L37 40"
                stroke="white"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </g>
            <defs>
              <filter
                id="filter0_d_61_98"
                x="0"
                y="0"
                width="84"
                height="84"
                filterUnits="userSpaceOnUse"
                color-interpolation-filters="sRGB"
              >
                <feFlood flood-opacity="0" result="BackgroundImageFix" />
                <feColorMatrix
                  in="SourceAlpha"
                  type="matrix"
                  values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                  result="hardAlpha"
                />
                <feOffset />
                <feGaussianBlur stdDeviation="15" />
                <feComposite in2="hardAlpha" operator="out" />
                <feColorMatrix
                  type="matrix"
                  values="0 0 0 0 0.431373 0 0 0 0 0.337255 0 0 0 0 0.811765 0 0 0 1 0"
                />
                <feBlend mode="normal" in2="BackgroundImageFix" result="effect1_dropShadow_61_98" />
                <feBlend
                  mode="normal"
                  in="SourceGraphic"
                  in2="effect1_dropShadow_61_98"
                  result="shape"
                />
              </filter>
            </defs>
          </svg>
          <p className="relative text-xs right-4">dom@keyauth.dev</p>
        </div>
      </div>
    </div>
  );
}

export function RateLimitsText() {
  return (
    <div className="flex flex-col text-white absolute left-[20px] sm:left-[40px] xl:left-[40px] bottom-[40px] max-w-[350px]">
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
            d="M21 4H18V3H21.5H22V3.5V8.42105V20.5V21H21.5H18V20H21V16H18V15H21V12H18V11H21V8.42105V8H18V7H21V4ZM9.85355 5.14645L9.5 4.79289L9.14645 5.14645L5.64645 8.64645L5.29289 9L5.64645 9.35355L6.79289 10.5L2.14645 15.1464L2 15.2929V15.5V17.5V18H2.5H5.5H6V17.5V16H7.5H8V15.5V14.7071L9.5 13.2071L10.6464 14.3536L11 14.7071L11.3536 14.3536L14.8536 10.8536L15.2071 10.5L14.8536 10.1464L9.85355 5.14645ZM7.85355 10.1464L7.5 9.79289L6.70711 9L9.5 6.20711L13.7929 10.5L11 13.2929L10.2071 12.5L9.85355 12.1464L7.85355 10.1464ZM3 15.7071L7.5 11.2071L8.79289 12.5L7.14645 14.1464L7 14.2929V14.5V15H5.5H5V15.5V17H3V15.7071ZM9.14645 9.35355L10.6464 10.8536L11.3536 10.1464L9.85355 8.64645L9.14645 9.35355Z"
            fill="white"
            fillOpacity="0.4"
          />
        </svg>
        <h3 className="ml-4 text-lg font-medium text-white">Rate Limits</h3>
      </div>
      <p className="mt-4 leading-6 text-white/60">
        Per IP, per user, per API key, or any identifier that matters to you. Enforced on the edge,
        as close to your users as possible, delivering fast and reliable rate limiting.
      </p>
    </div>
  );
}
