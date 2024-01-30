export function RateLimitsBento() {
  return (
    <div className="w-full mt-5 relative border-[.75px] h-[520px] rounded-[32px] border-[#ffffff]/20 flex overflow-x-hidden">
      <RateLimits />
      <RateLimitsText />
    </div>
  );
}

export function RateLimits() {
  return (
    <div className="mx-[40px] flex w-full flex-col">
      <div className="flex h-[200px] w-full  ratelimits-editor-bg-gradient rounded-b-xl">
        <div className="flex flex-col font-mono text-sm text-white px-[24px] space-y-3 mt-1 border-r-[.75px] border-[#ffffff]/20">
          <p>1</p>
          <p>2</p>
          <p>3</p>
          <p>4</p>
          <p>5</p>
          <p>6</p>
        </div>
        <div className="flex font-mono ratelimits-editor-bg-gradient-2 text-xs w-full text-white whitespace-pre leading-8 pl-8">
          {JSON.stringify({ rateLimit: { limit: 10, interval: 1000 } }, null, 2)}
        </div>
      </div>
      <div className="mt-8 flex flex-col">
        <div className="flex items-center">
          <div className="text-white font-mono text-sm">
            <span className="text-[#ffffff]/40">Creating</span> keys
            <span className="tracking-[-5px]">...</span>
            <span className="inline-flex w-[4px] h-[12px] bg-white ratelimits-bar-shadow ml-3" />
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
                    shape-rendering="crispEdges"
                  />
                  <rect
                    x="8"
                    y="6"
                    width="24"
                    height="24"
                    rx="6"
                    fill="black"
                    fill-opacity="0.15"
                    shape-rendering="crispEdges"
                  />
                  <rect
                    x="8.375"
                    y="6.375"
                    width="23.25"
                    height="23.25"
                    rx="5.625"
                    stroke="white"
                    stroke-opacity="0.1"
                    stroke-width="0.75"
                    shape-rendering="crispEdges"
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
            <p className="relative right-4 text-[13px]">sk_TEwCE9AY9BFTq1XJdIO</p>
          </div>
        </div>
        <div className="inline-flex w-[205px] opacity-70 items-center ml-[150px] mt-[30px] h-[36px] text-white font-mono text-sm ratelimits-link ratelimits-key-gradient border-[.75px] border-[#ffffff]/20 rounded-xl">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="62"
            height="36"
            viewBox="0 0 62 36"
            fill="none"
          >
            <g filter="url(#filter0_d_840_1934)">
              <rect
                x="8"
                y="6"
                width="24"
                height="24"
                rx="6"
                fill="#6E56CF"
                shape-rendering="crispEdges"
              />
              <rect
                x="8"
                y="6"
                width="24"
                height="24"
                rx="6"
                fill="black"
                fill-opacity="0.15"
                shape-rendering="crispEdges"
              />
              <rect
                x="8.375"
                y="6.375"
                width="23.25"
                height="23.25"
                rx="5.625"
                stroke="white"
                stroke-opacity="0.1"
                stroke-width="0.75"
                shape-rendering="crispEdges"
              />
              <path
                d="M21.5 15L23 16.5M14.5 23.5H17.5V21.5H19.5V20.5L21.5 18.5L19.5 16.5L14.5 21.5V23.5ZM18 15L23 20L26.5 16.5L21.5 11.5L18 15Z"
                stroke="white"
              />
            </g>
            <defs>
              <filter
                id="filter0_d_840_1934"
                x="-22"
                y="-24"
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
                <feBlend
                  mode="normal"
                  in2="BackgroundImageFix"
                  result="effect1_dropShadow_840_1934"
                />
                <feBlend
                  mode="normal"
                  in="SourceGraphic"
                  in2="effect1_dropShadow_840_1934"
                  result="shape"
                />
              </filter>
            </defs>
          </svg>
          <p className="relative right-4 text-[13px]">sk_UNWrXjYp6AF2h7Nx</p>
        </div>
      </div>
    </div>
  );
}

export function RateLimitsText() {
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
            d="M16.7 3C17.5483 3 18.1545 3.00039 18.6297 3.03921C19.099 3.07756 19.3963 3.15089 19.635 3.27248C20.1054 3.51217 20.4878 3.89462 20.7275 4.36502C20.8491 4.60366 20.9224 4.90099 20.9608 5.37032C20.9996 5.84549 21 6.45167 21 7.3V8H22V7.3V7.27781C22 6.45653 22 5.80955 21.9575 5.28889C21.9141 4.75771 21.8239 4.31413 21.6185 3.91103C21.283 3.25247 20.7475 2.71703 20.089 2.38148C19.6859 2.17609 19.2423 2.08593 18.7111 2.04253C18.1905 1.99999 17.5435 2 16.7222 2H16.7H16V3H16.7ZM7.27779 2H7.3H8V3H7.3C6.45167 3 5.84549 3.00039 5.37032 3.03921C4.90099 3.07756 4.60366 3.15089 4.36502 3.27248C3.89462 3.51217 3.51217 3.89462 3.27248 4.36502C3.15089 4.60366 3.07756 4.90099 3.03921 5.37032C3.00039 5.84549 3 6.45167 3 7.3V8H2V7.3V7.27779V7.27779C2 6.45652 1.99999 5.80954 2.04253 5.28889C2.08593 4.75771 2.17609 4.31414 2.38148 3.91103C2.71703 3.25247 3.25247 2.71703 3.91103 2.38148C4.31413 2.17609 4.75771 2.08593 5.28889 2.04253C5.80954 1.99999 6.45652 2 7.27778 2H7.27779ZM2 16.7V16H3V16.7C3 17.5483 3.00039 18.1545 3.03921 18.6297C3.07756 19.099 3.15089 19.3963 3.27248 19.635C3.51217 20.1054 3.89462 20.4878 4.36502 20.7275C4.60366 20.8491 4.90099 20.9224 5.37032 20.9608C5.8455 20.9996 6.45167 21 7.3 21H8V22H7.3H7.27781C6.45653 22 5.80955 22 5.28889 21.9575C4.75771 21.9141 4.31414 21.8239 3.91103 21.6185C3.25247 21.283 2.71703 20.7475 2.38148 20.089C2.17609 19.6859 2.08593 19.2423 2.04253 18.7111C1.99999 18.1905 2 17.5435 2 16.7222V16.7222V16.7ZM22 16V16.7V16.7222C22 17.5435 22 18.1905 21.9575 18.7111C21.9141 19.2423 21.8239 19.6859 21.6185 20.089C21.283 20.7475 20.7475 21.283 20.089 21.6185C19.6859 21.8239 19.2423 21.9141 18.7111 21.9575C18.1905 22 17.5435 22 16.7222 22H16.7H16V21H16.7C17.5483 21 18.1545 20.9996 18.6297 20.9608C19.099 20.9224 19.3963 20.8491 19.635 20.7275C20.1054 20.4878 20.4878 20.1054 20.7275 19.635C20.8491 19.3963 20.9224 19.099 20.9608 18.6297C20.9996 18.1545 21 17.5483 21 16.7V16H22ZM13.8536 5.14645L13.5 4.79289L13.1464 5.14645L9.64645 8.64645L9.29289 9L9.64645 9.35355L10.7929 10.5L6.14645 15.1464L6 15.2929V15.5V17.5V18H6.5H9.5H10V17.5V16H11.5H12V15.5V14.7071L13.5 13.2071L14.6464 14.3536L15 14.7071L15.3536 14.3536L18.8536 10.8536L19.2071 10.5L18.8536 10.1464L13.8536 5.14645ZM11.8536 10.1464L11.5 9.79289L10.7071 9L13.5 6.20711L17.7929 10.5L15 13.2929L14.2071 12.5L13.8536 12.1464L11.8536 10.1464ZM7 15.7071L11.5 11.2071L12.7929 12.5L11.1464 14.1464L11 14.2929V14.5V15H9.5H9V15.5V17H7V15.7071ZM13.1464 9.35355L14.6464 10.8536L15.3536 10.1464L13.8536 8.64645L13.1464 9.35355Z"
            fill="white"
            fill-opacity="0.4"
          />
        </svg>
        <h3 className="text-lg font-medium text-white ml-4">Rate Limits</h3>
      </div>
      <p className="mt-4 text-white/60 leading-6">
        Implement granular control over access with per-key rate limiting, preventing abuse and
        optimizing the performance of your services.
      </p>
    </div>
  );
}
