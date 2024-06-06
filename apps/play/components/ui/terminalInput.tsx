interface TerminalInputProps extends React.ComponentPropsWithoutRef<"div"> {
  sendInput: (cmd: string) => void;
  classNames?: {
    header?: string;
    input?: string;
    frame?: string;
  };
}
import { cn } from "@/lib/utils";
import { GeistMono } from "geist/font/mono";
import { useState } from "react";
export default function TerminalInput({ sendInput }: TerminalInputProps) {
  const cols = 150;
  const [rows, setRows] = useState(1);
  function handleInput(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    sendInput(event.currentTarget.input.value);
    event.currentTarget.input.value = "";
    setRows(1);
  }
  function keyPressed(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter") {
      sendInput(e.currentTarget.value);
      e.currentTarget.value = "";
      setRows(1);
    }
    const lines = e.currentTarget.value.split("\n");
    if (e.currentTarget.value.length > cols - 10) {
      const temp = Math.ceil(e.currentTarget.value.length / cols);
      if (lines.length > temp) {
        setRows(lines.length);
      } else {
        setRows(temp);
      }
    }
  }

  return (
    <div className={"flex w-full bg-[#1F1F1E]/70 border border-white/30 "}>
      <label className="animate-pulse pl-4 mt-2 text-xl text-white">{">>>"}</label>
      <form onSubmit={handleInput}>
        <textarea
          cols={cols}
          rows={rows}
          wrap="hard"
          name="input"
          onKeyUp={(e) => keyPressed(e)}
          className={cn(
            "w-full bg-transparent h-full text-white border-hidden focus:outline-none whitespace-wrap p-2 scrollbar-hide",
            GeistMono.className,
          )}
          placeholder=""
        />
      </form>
    </div>
  );
}
