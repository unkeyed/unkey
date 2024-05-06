import { cn } from "@/lib/utils";
import s from "./accelerate.module.css";

export default function AcceleratePage() {
  return (
    <div className="container min-h-[100dvh] flex flex-col justify-between">
      <header className="flex w-full justify-between items-center h-24">
        <div>Left</div>
        <div>Right</div>
      </header>

      <div className="w-full flex items-center justify-center">
        {/* Middle Canvas */}
        <div className="relative w-[600px] aspect-square">
          {/* Outer Circle */}
          <div className={cn("absolute inset-0 border border-white rounded-full", s.outer)} />
          {/* Pointer Controller */}
          <div
            className={cn(
              "absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-px h-px bg-[green] rotate-[--pointer-rotation]",
              s.pointer,
            )}
          >
            <div className="relative h-[300px] aspect-[53/1097] -translate-x-1/2 -translate-y-[97.72%]">
              <Pointer />
            </div>
          </div>
          {/* Inner Circle */}
          <div className="relative top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[260px] aspect-square rounded-full bg-black/80 border border-white backdrop-blur-[2px]" />

          {/* <div className="relative [transform-origin:49.05660377%_97.72209567%]">
            <Pointer />
          </div> */}
        </div>
      </div>

      <footer className="flex w-full justify-between items-center h-24">
        <div>Left</div>
        <div>Right</div>
      </footer>
    </div>
  );
}

function Pointer() {
  return (
    <svg
      width="100%"
      height="100%"
      viewBox="0 0 53 1097"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M26.5 0L53 1077C53 1088.05 41.1355 1097 26.5 1097C11.8645 1097 0 1088.05 0 1077L26.5 0Z"
        fill="#D9D9D9"
      />
    </svg>
  );
}
