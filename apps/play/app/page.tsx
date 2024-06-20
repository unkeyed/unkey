import { Button } from "@/components/ui/button";
import { NamedInput } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import Link from "next/link";
import React from "react";

// TODO: move this to a react state
const curlEquivalent = `curl --request POST \n   --url https://api.unkey.dev/v1/apis.createApi \n   --header 'Authorization: Bearer <token>' \n   --header 'Content-Type: application/json' \n   --data '{   "name": "my-untitled-api" }'`;

export default function Page() {
  return (
    <div className="w-full h-full min-h-[100dvh] text-sm text-[#E2E2E2] flex flex-col">
      <header className="w-full flex grow-0 shrink-0 justify-center items-center lg:border-b h-14 text-[#fff] text-xl">
        <nav className="w-full max-w-[500px] h-full px-5 flex items-center justify-between">
          <Link href="https://unkey.dev" target="_blank" className="flex items-center text gap-3">
            <h1>
              <SVGLogoUnkey />
            </h1>
            <span>/</span>
            <h1>Playground</h1>
          </Link>

          <Button asChild>
            <Link href="https://app.unkey.com/auth/sign-up">Try Unkey</Link>
          </Button>
        </nav>
      </header>

      <div className="w-full flex flex-col items-center flex-1 h-full max-w-[500px] pb-5 mx-auto">
        {/* Left panel */}
        <div className="w-full flex flex-col grow-0 shrink-0 px-5 h-[298px]">
          <span className="uppercase text-[#A1A1A1]">Introduction</span>

          <div className="mt-3.5">
            Let's get started!
            <br />
            <br />
            Register your first API by calling the official Unkey endpoint.
            <br />
            <br />
            Name your API below, then send the request to our endpoint!
          </div>
        </div>

        {/* Right panel */}
        <div className="w-full pt-2.5 flex flex-col flex-1 px-5 justify-between gap-2 bg-[#080808] border-t-2 border-[#212121]">
          <div className="flex flex-col w-full">
            <span className="uppercase text-[#A1A1A1]">Select Unkey Endpoint:</span>

            <Tabs defaultValue="createApi" className="w-full mt-3">
              <TabsList className="w-full font-mono overflow-x-scroll max-w-fit">
                {/* <TabsTrigger value="none">
                  None
                </TabsTrigger> */}
                <TabsTrigger value="createApi">POST /apis.createApi</TabsTrigger>
              </TabsList>
              {/* <TabsContent className="mt-3" value="none">
                You haven't selected any endpoints.
                Please select an endpoint to continue!
              </TabsContent> */}
              <TabsContent className="mt-3" value="createApi">
                <form action="" className="w-full flex flex-col">
                  <legend>You'll call the endpoint with variables:</legend>

                  <NamedInput
                    label="name"
                    placeholder="my-untitled-api"
                    className="mt-2 font-mono"
                  />
                </form>
              </TabsContent>
            </Tabs>
          </div>

          <div className="flex flex-col w-full">
            <Button>Send Request</Button>

            <label className="mt-3">Equivalent CURL request:</label>
            <Textarea
              className="mt-2 resize-none font-mono h-[104px] text-xs p-3 text-[#686868]"
              value={curlEquivalent}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

// function Container({ className, ...props }: React.HTMLProps<HTMLDivElement>) {
//   return <div className="w-full max-w-[640px]" {...props} />;
// }

function SVGLogoUnkey() {
  return (
    <svg width="59" height="18" viewBox="0 0 59 18" fill="none" xmlns="http://www.w3.org/2000/svg">
      <title>Unkey</title>

      <path
        d="M6.05571 14.0143C2.19857 14.0143 0 11.97 0 8.46003V0.900024H2.06357V8.32503C2.06357 10.9286 3.22071 12.0086 6.05571 12.0086C8.89071 12.0086 10.0479 10.9286 10.0479 8.32503V0.900024H12.1307V8.46003C12.1307 11.97 9.93214 14.0143 6.05571 14.0143ZM16.0593 13.8215H13.9764V4.23645H15.8857V7.20645H16.0207C16.31 5.58645 17.5828 4.0436 20.0128 4.0436C22.6743 4.0436 23.9857 5.83717 23.9857 8.05503V13.8215H21.9028V8.61431C21.9028 6.82074 21.0928 5.91431 19.1064 5.91431C17.0043 5.91431 16.0593 6.99431 16.0593 9.07717V13.8215ZM27.9245 13.8215H25.8416V0.900024H27.9245V8.03574H30.6631L33.5366 4.23645H35.9666L32.3602 8.84574L35.9473 13.8215H33.498L30.6631 9.90645H27.9245V13.8215ZM41.8126 14.0143C38.6691 14.0143 36.6055 12.24 36.6055 9.0386C36.6055 6.04931 38.6498 4.0436 41.7741 4.0436C44.7441 4.0436 46.7691 5.68288 46.7691 8.59503C46.7691 8.94217 46.7498 9.21217 46.6919 9.50145H38.5533C38.6305 11.3529 39.5369 12.3365 41.7548 12.3365C43.7605 12.3365 44.5898 11.6807 44.5898 10.5429V10.3886H46.6726V10.5622C46.6726 12.6065 44.6669 14.0143 41.8126 14.0143ZM41.7355 5.68288C39.6141 5.68288 38.6883 6.62788 38.5726 8.34431H44.8019V8.30574C44.8019 6.53145 43.7798 5.68288 41.7355 5.68288ZM49.8719 17.1H48.5027V15.21H50.3734C51.2219 15.21 51.5691 14.9786 51.8584 14.3229L52.0898 13.8215L47.3648 4.23645H49.6984L52.1477 9.32788L53.0927 11.6229H53.2469L54.1534 9.3086L56.4098 4.23645H58.7048L53.7098 14.8629C52.9191 16.5793 51.8391 17.1 49.8719 17.1Z"
        fill="white"
      />
    </svg>
  );
}
