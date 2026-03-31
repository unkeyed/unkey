"use client";
import { Github, Layers3 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { OnboardingLinks } from "../onboarding-links";

const FlyioLogo = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 167 151"
    fill="currentColor"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path d="M82.348 6.956l.079-.006v68.484l-.171-.315a191.264 191.264 0 01-6.291-12.75 136.318 136.318 0 01-4.269-10.688 84.358 84.358 0 01-2.574-8.802c-.541-2.365-.956-4.765-1.126-7.19a35.028 35.028 0 01-.059-3.108c.016-.903.053-1.804.109-2.705.09-1.418.234-2.832.442-4.235.165-1.104.368-2.205.62-3.293.2-.865.431-1.723.696-2.567.382-1.22.84-2.412 1.373-3.576.195-.419.405-.836.624-1.245 1.322-2.449 3.116-4.704 5.466-6.214a11.422 11.422 0 015.081-1.79zm8.88.173l4.607 1.314a28.193 28.193 0 016.076 3.096 24.387 24.387 0 016.533 6.517 24.618 24.618 0 012.531 4.878 28.586 28.586 0 011.761 7.898c.061.708.096 1.418.11 2.127.016.659.012 1.321-.041 1.98a22.306 22.306 0 01-.828 4.352 34.281 34.281 0 01-1.194 3.426 49.43 49.43 0 01-1.895 4.094c-1.536 2.966-3.304 5.803-5.195 8.547a133.118 133.118 0 01-7.491 9.776 185.466 185.466 0 01-8.987 9.96c2.114-3.963 4.087-8 5.915-12.102a149.96 149.96 0 002.876-6.93 108.799 108.799 0 002.679-7.792 76.327 76.327 0 001.54-5.976c.368-1.727.657-3.472.836-5.228.15-1.464.205-2.937.169-4.406a62.154 62.154 0 00-.1-2.695c-.216-3.612-.765-7.212-1.818-10.676a31.255 31.255 0 00-1.453-3.849c-1.348-2.937-3.23-5.683-5.776-7.686l-.855-.625z" />
    <path d="M78.453 82.687l-2.474-2.457-1.497-1.55-3.779-4.066-.942-1.031-4.585-5.36-1.297-1.609-4.015-5.225-.446-.602-3.086-4.544-.846-1.369-2.211-3.795-.675-1.241-1.912-4.04-.362-.927-1.098-3.092-.423-1.46-.326-1.349-.275-1.465-.215-1.627-.088-1.039-.036-1.774.008-.372.051-1.062.382-3.869.12-.677.871-3.862.201-.647.647-1.886.207-.488 1.03-2.262.714-1.346.994-1.64.991-1.46.706-.928.813-.98.895-.985.767-.771 1.867-1.643 1.365-1.117c.033-.028.067-.053.102-.077l1.615-1.092 1.283-.818L65.931 3.8c.037-.023.079-.041.118-.059l3.456-1.434.319-.12 3.072-.899 1.297-.291 1.754-.352L77.11.468l1.784-.222L80.11.138 82.525.01l.946-.01 1.791.037.466.026 2.596.216 3.433.484.397.083 3.393.844.996.297 1.107.383 1.348.51 1.066.452 1.566.738.987.507 1.774 1.041.661.407 2.418 1.765.694.602 1.686 1.536.083.083 1.43 1.534.492.555 1.678 2.23.342.533 1.332 2.249.401.771.751 1.678.785 1.959.279.82.809 2.949c.015.052.027.105.037.159l.63 3.988.126 1.384.102 1.781.01.371-.038 1.989-.033.527-.108.86-.555 3.177-.134.582-1.28 3.991a1.186 1.186 0 01-.04.114l-1.188 2.876-.045.095-1.552 3.1-2.713 4.725-1.44 2.203-1.729 2.585-1.219 1.67-2.414 3.228-1.644 2.067-2.428 2.957-1.703 1.992-2.618 2.945-1.684 1.849-4.869 5.085-1.133 1.119.669.569c.946.871 1.835 1.8 2.661 2.787.248.301.488.608.72.921.506.685.962 1.406 1.362 2.158.216.407.409.828.58 1.257.389.985.651 2.026.749 3.078l.044.799c.025 1.53-.255 3.05-.823 4.471a11.057 11.057 0 01-3.479 4.625c-.541.424-1.118.796-1.724 1.117a12.347 12.347 0 01-4.516 1.341h-.01a12.996 12.996 0 01-5.476-.623 11.933 11.933 0 01-2.319-1.096 11.268 11.268 0 01-2.329-1.896 11.06 11.06 0 01-2.209-3.464 11.468 11.468 0 01-.819-3.972l.014-.966c.073-1.119.315-2.221.718-3.267.157-.411.334-.812.531-1.202.386-.755.83-1.477 1.324-2.164.323-.45.667-.887 1.025-1.31a30.309 30.309 0 012.384-2.49l.309-.279.497-.415z" />
    <path d="M116.78 20.613h19.23c17.104 0 30.99 13.886 30.99 30.99v67.618c0 17.104-13.886 30.99-30.99 30.99h-1.516c-8.803-1.377-12.621-4.017-15.57-6.248L94.475 123.86a3.453 3.453 0 00-4.329 0l-7.943 6.532-22.37-18.394a3.443 3.443 0 00-4.326 0l-31.078 27.339c-6.255 5.087-10.392 4.148-13.075 3.853C4.424 137.502 0 128.874 0 119.221V51.603c0-17.104 13.886-30.99 30.993-30.99H50.18l-.035.077-.647 1.886-.201.647-.871 3.862-.12.677-.382 3.869-.051 1.062-.008.372.036 1.774.088 1.039.215 1.627.275 1.465.326 1.349.423 1.46 1.098 3.092.362.927 1.912 4.04.675 1.241 2.211 3.795.846 1.369 3.086 4.544.446.602 4.015 5.225 1.297 1.609 4.585 5.36.942 1.031 3.779 4.066 1.497 1.55 2.474 2.457-.497.415-.309.279a30.309 30.309 0 00-2.384 2.49c-.359.423-.701.86-1.025 1.31-.495.687-.938 1.41-1.324 2.164-.198.391-.375.792-.531 1.202a11.098 11.098 0 00-.718 3.267l-.014.966c.035 1.362.312 2.707.819 3.972a11.06 11.06 0 002.209 3.464 11.274 11.274 0 002.329 1.896c.731.447 1.51.815 2.319 1.096 1.76.597 3.627.809 5.476.623h.01a12.347 12.347 0 004.516-1.341 11.573 11.573 0 001.724-1.117 11.057 11.057 0 003.479-4.625c.569-1.422.848-2.941.823-4.471l-.044-.799a11.305 11.305 0 00-.749-3.078c-.17-.429-.364-.848-.58-1.257-.4-.752-.856-1.473-1.362-2.158-.232-.313-.472-.62-.72-.921a29.81 29.81 0 00-2.661-2.787l-.669-.569 1.133-1.119 4.869-5.085 1.684-1.849 2.618-2.945 1.703-1.992 2.428-2.957 1.644-2.067 2.414-3.228 1.219-1.67 1.729-2.585 1.44-2.203 2.713-4.725 1.552-3.1.045-.095 1.188-2.876c.015-.037.029-.076.04-.114l1.28-3.991.134-.582.555-3.177.108-.86.033-.527.038-1.989-.01-.371-.102-1.781-.126-1.384-.63-3.988a1.521 1.521 0 00-.037-.159l-.809-2.949-.279-.82-.364-.907zm9.141 84.321c-4.007.056-7.287 3.336-7.343 7.342.059 4.006 3.337 7.284 7.343 7.341 4.005-.058 7.284-3.335 7.345-7.341-.058-4.006-3.338-7.286-7.345-7.342z" fillOpacity=".35" />
    <path d="M71.203 148.661l19.927-16.817a2.035 2.035 0 012.606-.006l20.216 16.823a6.906 6.906 0 004.351 1.55H66.877a6.805 6.805 0 004.326-1.55zm12.404-60.034l.195.057c.063.03.116.075.173.114l.163.144c.402.37.793.759 1.169 1.157.265.283.523.574.771.875.315.38.61.779.879 1.194.116.183.224.368.325.561.088.167.167.34.236.515.122.305.214.627.242.954l-.006.614a3.507 3.507 0 01-1.662 2.732 4.747 4.747 0 01-2.021.665l-.759.022-.641-.056a4.964 4.964 0 01-.881-.214 4.17 4.17 0 01-.834-.391l-.5-.366a3.431 3.431 0 01-1.139-1.952 5.016 5.016 0 01-.059-.387l-.018-.586c.01-.158.034-.315.069-.472.087-.341.213-.673.372-.988.205-.396.439-.776.7-1.137.433-.586.903-1.143 1.405-1.67.324-.342.655-.673 1.001-.993l.246-.221c.171-.114.173-.114.368-.171h.206z" />
  </svg>
);

const VercelLogo = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="currentColor"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path d="m12 1.608 12 20.784H0Z" />
  </svg>
);

const CloudflareLogo = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="currentColor"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path d="M16.5088 16.8447c.1475-.5068.0908-.9707-.1553-1.3154-.2246-.3164-.6045-.499-1.0615-.5205l-8.6592-.1123a.1559.1559 0 0 1-.1333-.0713c-.0283-.042-.0351-.0986-.021-.1553.0278-.084.1123-.1484.2036-.1562l8.7359-.1123c1.0351-.0489 2.1601-.8868 2.5537-1.9136l.499-1.3013c.0215-.0561.0293-.1128.0147-.168-.5625-2.5463-2.835-4.4453-5.5499-4.4453-2.5039 0-4.6284 1.6177-5.3876 3.8614-.4927-.3658-1.1187-.5625-1.794-.499-1.2026.119-2.1665 1.083-2.2861 2.2856-.0283.31-.0069.6128.0635.894C1.5683 13.171 0 14.7754 0 16.752c0 .1748.0142.3515.0352.5273.0141.083.0844.1475.1689.1475h15.9814c.0909 0 .1758-.0645.2032-.1553l.12-.4268zm2.7568-5.5634c-.0771 0-.1611 0-.2383.0112-.0566 0-.1054.0415-.127.0976l-.3378 1.1744c-.1475.5068-.0918.9707.1543 1.3164.2256.3164.6055.498 1.0625.5195l1.8437.1133c.0557 0 .1055.0263.1329.0703.0283.043.0351.1074.0214.1562-.0283.084-.1132.1485-.204.1553l-1.921.1123c-1.041.0488-2.1582.8867-2.5527 1.914l-.1406.3585c-.0283.0713.0215.1416.0986.1416h6.5977c.0771 0 .1474-.0489.169-.126.1122-.4082.1757-.837.1757-1.2803 0-2.6025-2.125-4.727-4.7344-4.727" />
  </svg>
);

const RailwayLogo = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 1024 1024"
    fill="currentColor"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path d="M4.756 438.175A520.713 520.713 0 0 0 0 489.735h777.799c-2.716-5.306-6.365-10.09-10.045-14.772-132.97-171.791-204.498-156.896-306.819-161.26-34.114-1.403-57.249-1.967-193.037-1.967-72.677 0-151.688.185-228.628.39-9.96 26.884-19.566 52.942-24.243 74.14h398.571v51.909H4.756Z" />
    <path d="M783.93 541.696H.399c.82 13.851 2.112 27.517 3.978 40.999h723.39c32.248 0 50.299-18.297 56.162-40.999ZM45.017 724.306S164.941 1018.77 511.46 1024c207.112 0 385.071-123.006 465.907-299.694H45.017Z" />
    <path d="M511.454 0C319.953 0 153.311 105.16 65.31 260.612c68.771-.144 202.704-.226 202.704-.226h.031v-.051c158.309 0 164.193.707 195.118 1.998l19.149.706c66.7 2.224 148.683 9.384 213.19 58.19 35.015 26.471 85.571 84.896 115.708 126.52 27.861 38.499 35.876 82.756 16.933 125.158-17.436 38.97-54.952 62.215-100.383 62.215H16.69s4.233 17.944 10.58 37.751h970.632A510.385 510.385 0 0 0 1024 512.218C1024.01 229.355 794.532 0 511.454 0Z" />
  </svg>
);

type ConnectGithubStepProps = {
  projectId: string;
  onBeforeNavigate?: () => void;
};

export const ConnectGithubStep = ({ projectId, onBeforeNavigate }: ConnectGithubStepProps) => {
  const installUrl = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(JSON.stringify({ projectId }))}`;

  return (
    <div className="flex flex-col items-center">
      <div className="flex flex-col gap-3 min-w-[600px]">
        <div className="border border-grayA-5 rounded-[14px] flex justify-start items-center gap-4 py-[18px] px-4">
          <div className="size-8 rounded-[10px] bg-gray-12 grid place-items-center">
            <Layers3 className="size-[18px] text-gray-1" iconSize="md-medium" />
          </div>
          <div className="flex flex-col gap-3">
            <span className="font-medium text-gray-12 text-[13px] leading-[9px]">
              Import from GitHub
            </span>
            <span className="text-gray-10 text-[13px] leading-[9px]">
              Add a repo from your GitHub account
            </span>
          </div>
          <a
            href={installUrl}
            rel="noopener noreferrer"
            className="ml-auto"
            onClick={onBeforeNavigate}
          >
            <Button
              variant="outline"
              className="rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
            >
              <Github className="size-[18px]! text-gray-12 shrink-0" />
              <span className="text-[13px] text-gray-12 font-medium">Connect GitHub</span>
            </Button>
          </a>
        </div>

        <div className="border border-grayA-5 rounded-[14px] flex justify-start items-center gap-4 py-[18px] px-4">
          <div className="size-8 rounded-[10px] bg-[#000] dark:bg-white grid place-items-center">
            <RailwayLogo className="size-[18px] text-white dark:text-black" />
          </div>
          <div className="flex flex-col gap-3">
            <span className="font-medium text-gray-12 text-[13px] leading-[9px]">
              Import from Railway
            </span>
            <span className="text-gray-10 text-[13px] leading-[9px]">
              Migrate an existing project from Railway
            </span>
          </div>
          <div className="ml-auto">
            <Button
              variant="outline"
              className="rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
              disabled
            >
              <RailwayLogo className="size-[18px]! text-gray-12 shrink-0" />
              <span className="text-[13px] text-gray-12 font-medium">Migrate from Railway</span>
            </Button>
          </div>
        </div>

        <div className="border border-grayA-5 rounded-[14px] flex justify-start items-center gap-4 py-[18px] px-4">
          <div className="size-8 rounded-[10px] bg-[#24175b] grid place-items-center">
            <FlyioLogo className="size-[18px] text-white" />
          </div>
          <div className="flex flex-col gap-3">
            <span className="font-medium text-gray-12 text-[13px] leading-[9px]">
              Import from Fly.io
            </span>
            <span className="text-gray-10 text-[13px] leading-[9px]">
              Migrate an existing app from Fly.io
            </span>
          </div>
          <div className="ml-auto">
            <Button
              variant="outline"
              className="rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
              disabled
            >
              <FlyioLogo className="size-[18px]! text-gray-12 shrink-0" />
              <span className="text-[13px] text-gray-12 font-medium">Migrate from Fly.io</span>
            </Button>
          </div>
        </div>

        <div className="border border-grayA-5 rounded-[14px] flex justify-start items-center gap-4 py-[18px] px-4">
          <div className="size-8 rounded-[10px] bg-black grid place-items-center">
            <VercelLogo className="size-[14px] text-white" />
          </div>
          <div className="flex flex-col gap-3">
            <span className="font-medium text-gray-12 text-[13px] leading-[9px]">
              Import from Vercel
            </span>
            <span className="text-gray-10 text-[13px] leading-[9px]">
              Migrate an existing project from Vercel
            </span>
          </div>
          <div className="ml-auto">
            <Button
              variant="outline"
              className="rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
              disabled
            >
              <VercelLogo className="size-[14px]! text-gray-12 shrink-0" />
              <span className="text-[13px] text-gray-12 font-medium">Migrate from Vercel</span>
            </Button>
          </div>
        </div>

        <div className="border border-grayA-5 rounded-[14px] flex justify-start items-center gap-4 py-[18px] px-4">
          <div className="size-8 rounded-[10px] bg-[#F48120] grid place-items-center">
            <CloudflareLogo className="size-[18px] text-white" />
          </div>
          <div className="flex flex-col gap-3">
            <span className="font-medium text-gray-12 text-[13px] leading-[9px]">
              Import from Cloudflare
            </span>
            <span className="text-gray-10 text-[13px] leading-[9px]">
              Migrate Workers from Cloudflare
            </span>
          </div>
          <div className="ml-auto">
            <Button
              variant="outline"
              className="rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
              disabled
            >
              <CloudflareLogo className="size-[18px]! text-gray-12 shrink-0" />
              <span className="text-[13px] text-gray-12 font-medium">Migrate from Cloudflare</span>
            </Button>
          </div>
        </div>
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
    </div>
  );
};
