"use client";

import TerminalInput from "@/components/ui/terminalInput";
import { type Message, getStepsData } from "@/lib/data";
import { handleCurlServer } from "@/lib/helper";
import { cn } from "@/lib/utils";
import { GeistMono } from "geist/font/mono";
import { KeyRound, SquareArrowOutUpRight } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { TypeAnimation } from "react-type-animation";

export default function Home() {
  const data = getStepsData();
  const apiId = process.env.NEXT_PUBLIC_PLAYGROUND_API_ID;
  const [historyItems, setHistoryItems] = useState<Message[]>(data ? data[0].messages : []);
  const step = useRef<number>(0);
  const timeStamp = useRef<number>(Date.now() + 24 * 60 * 60 * 1000);
  const keyId = useRef<string>();
  const keyName = useRef<string>();
  const scrollRef = useRef<null | HTMLDivElement>(null);

  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  const parseCurlCommand = useCallback(
    (stepString: string) => {
      let tempString = stepString;
      tempString = tempString.replace("<timeStamp>", timeStamp.current.toString());
      if (apiId) {
        tempString = apiId.length > 0 ? tempString.replace("<apiId>", apiId) : tempString;
      }
      tempString = keyId.current ? tempString.replace("<keyId>", keyId.current) : tempString;
      tempString = keyName.current ? tempString.replace("<key>", keyName.current) : tempString;
      return tempString;
    },
    [apiId],
  );
  function handleSubmit(cmd: string) {
    postNewLine(cmd, "text-violet-500");
    handleCurl(cmd);
  }

  function postNewLine(input: string, color: string) {
    const temp = historyItems;
    temp.push({ content: input, color: color });
    setHistoryItems([...temp]);
  }

  async function handleCurl(curlString: string) {
    postNewLine("Processing...", "text-green-500");
    if (!curlString.includes("curl")) {
      postNewLine('{"Error", "Invalid Curl Command"}', "text-red-500");
      return;
    }
    const parsedCurlString = curlString.replace("--data", "--data-raw");
    const response = await handleCurlServer(parsedCurlString);
    if (response) {
      const resJson = JSON.parse(JSON.stringify(response, null, 2));
      if (resJson.error) {
        postNewLine(JSON.stringify(response, null, 2), "text-red-500");
        return;
      }
      const result = resJson.result;
      // Response from server to Terminal
      postNewLine(JSON.stringify(result, null, 2), "text-blue-500");

      if (result.keyId) {
        keyId.current = result.keyId;
      }
      if (result.key) {
        keyName.current = result.key;
      }

      const newCurl = parseCurlCommand(data[step.current + 1].curlCommand ?? "");
      postNewLine(data[step.current + 1].header, "text-white");
      const newMessages = data[step.current + 1].messages;
      newMessages.map((item) => {
        const cmd = parseCurlCommand(item.content);
        postNewLine(cmd, "text-white");
      });
      postNewLine(newCurl, "text-white");
      step.current += 1;
    }
  }

  const HistoryList = () => {
    return historyItems?.map((item, index) => {
      {
        const isLast = index === historyItems.length - 1;
        const isCurl = item.content.includes("curl --request");
        if (isLast) {
          return (
            <button
              className={"mt-2 text-left text-wrap"}
              type="button"
              onClick={() => handleSubmit(item.content)}
            >
              <TypeAnimation
                key={`curl${index.toString()}`}
                className={item.color !== "undefined" ? "text-white" : ""}
                sequence={[item.content]}
                speed={99}
                repeat={0}
              />
            </button>
          );
        }
        if (!isLast && isCurl) {
          return (
            <p
              key={`curl${index.toString()}`}
              className={cn(
                "font-medium leading-7 mt-4 text-wrap",
                item.color ?? "text-white",
                GeistMono.className,
              )}
            >
              {item.content}
            </p>
          );
        }
        return (
          <p
            key={`curl${index.toString()}`}
            className={cn(
              "font-medium leading-7 text-wrap mt-4",
              item.color ?? "text-white",
              GeistMono.className,
            )}
          >
            {item.content}
          </p>
        );
      }
    });
  };
  return (
    <div className="flex flex-col w-full h-full justify-center px-2">
      {/* Desktop */}
      <div className="mx-auto w-full h-full justify-center max-w-[1440px] hidden md:flex flex-col px-4">
        <div className=" min-w-full h-full mt-12">
          <div className="flex flex-row w-full h-8 bg-[#383837]/60 rounded-t-lg drop-shadow-[0_2px_1px_rgba(0,0,0,0.7)]">
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
          <div className="flex flex-col min-w-full h-[900px] bg-[#1F1F1E]/80 overflow-hidden ">
            <div
              onChange={() => scrollTo()}
              className="flex flex-col w-full rounded-lg pt-4 pl-6 scrollbar-hide overflow-y-scroll scroll-smooth snap-y pr-24"
            >
              <HistoryList />
              <div ref={scrollRef} />
            </div>
          </div>
          <TerminalInput sendInput={(cmd) => handleSubmit(cmd)} />
        </div>
      </div>
      {/* Mobile */}
      <div className="relative min-h-screen max-h-screen justify-between md:hidden ">
        <div className="flex-grow overflow-y-hidden px-2 h-full">
          <HistoryList />
          <div ref={scrollRef} className="h-4">
            {""}
          </div>
        </div>
        <div className="relative bottom-0 justify-end w-full mt-2">
          <TerminalInput sendInput={(cmd) => handleSubmit(cmd)} />
        </div>
      </div>
    </div>
  );
}
