import { GeistMono } from "geist/font/mono";
import { KeyRound, SquareArrowOutUpRight } from "lucide-react";
import { useEffect, useState } from "react";

import type { Message, Step } from "@/lib/playground/data";
import { cn } from "@/lib/utils";

import TerminalInput from "./terminalInput";
import TextAnimator from "./textAnimator";

interface TerminalProps extends React.ComponentPropsWithoutRef<"div"> {
  sendRequest: (curl: string) => void;
  response: string;
  stepData: Step;
}

export default function Terminal({ sendRequest, response, stepData }: TerminalProps) {
  const historyItems: Message[] = [];
  const [curlResponse, setCurlResponse] = useState<string>(response);

  useEffect(() => {
    stepData.messages.map((item: Message) => {
      historyItems.push({
        content: item.content,
        color: "text-white-600",
      });
    });
  }, [stepData]);

  useEffect(() => {
    if (curlResponse) {
      historyItems.push({
        content: response,
        color: "text-green-600",
      });
    }
  }, [curlResponse]);

  useEffect(() => {
    setCurlResponse(response);
  }, [response]);

  const HistoryList = () => {
    return historyItems?.map((item, index) => {
      if (index === historyItems.length - 1) {
        return (
          <div key={`cmd${index.toString()}`} className={cn("flex flex-row", GeistMono.className)}>
            <TextAnimator
              input={item.content}
              repeat={0}
              style={"flex flex-row font-medium leading-7 text-white"}
            />
          </div>
        );
      }
      return (
        <div key={`cmd${index.toString()}`} className={cn("flex flex-row  ", GeistMono.className)}>
          <p className={cn("flex flex-row font-light leading-7", item.color)}>{item.content}</p>
        </div>
      );
    });
  };

  async function handleSubmit(cmd: string) {
    // postNewLine(cmd);
    sendRequest(cmd);
  }

  return (
    <div className=" min-w-full h-screen">
      <div className="flex flex-row w-full h-8 bg-gray-900 rounded-t-lg drop-shadow-[0_2px_1px_rgba(0,0,0,0.7)]">
        <div className="flex flex-col w-1/3">
          <KeyRound size={18} className="mx-2 mt-1" />
        </div>
        <div className="flex flex-col w-1/3 justify-center">
          <p className="text-center">Heading</p>
        </div>
        <div className="flex flex-row w-1/3 justify-end my-auto">
          Step <SquareArrowOutUpRight size={18} className="pt-1 mx-2" />
        </div>
      </div>
      <div className="flex flex-col min-w-full h-full bg-gray-900 rounded-b-lg">
        <div className="flex flex-col w-full rounded-lg overflow-y-auto pt-4 pl-6">
          <HistoryList />
        </div>
        <TerminalInput sendInput={(cmd) => handleSubmit(cmd)} />
      </div>
    </div>
  );
}
