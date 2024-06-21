"use client";

import Link from "next/link";
import React, { type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { NamedInput } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { protectedApiRequestSchema } from "@/lib/schemas";

// TODO: move this to a react state
const curlEquivalent = `curl --request POST \n   --url https://api.unkey.dev/v1/apis.createApi \n   --header 'Authorization: Bearer <token>' \n   --header 'Content-Type: application/json' \n   --data '{   "name": "my-untitled-api" }'`;

function getBaseUrl() {
  if (typeof window !== "undefined") {
    // browser should use relative path
    return "";
  }

  if (process.env.NEXT_PUBLIC_VERCEL_URL) {
    // reference for vercel.com
    return `https://${process.env.NEXT_PUBLIC_VERCEL_URL}`;
  }

  // assume localhost
  return `http://localhost:${process.env.PORT ?? 3000}`;
}

const API_UNKEY_DEV_V1 = "https://api.unkey.dev/v1/";

export default function Page() {
  // const [isFreeMode, setIsFreeMode] = React.useState(false); // TODO: free mode in the future
  const [stepIdx, setStepIdx] = React.useState(0);
  const [lastResponseJson, setLastResponseJson] = React.useState<string | null>(null);
  const willUpdateCacheRef = React.useRef<boolean>(true);
  const cacheParsed = React.useMemo(() => {
    if (!willUpdateCacheRef.current) {
      return null;
    }

    const parsed = lastResponseJson !== null ? JSON.parse(lastResponseJson) : null;

    return parsed;
  }, [lastResponseJson]);

  const ALL_ENDPOINTS = React.useMemo(() => {
    return {
      "apis.createApi": {
        method: "POST",
        route: "apis.createApi",
        prefixUrl: API_UNKEY_DEV_V1,
        defaultValues: {
          name: "my-untitled-api",
        },
        mockedRequest: () => {
          return JSON.stringify({ apiId: process.env.NEXT_PUBLIC_PLAYGROUND_API_ID });
        },
      },
      "keys.createKey": {
        method: "POST",
        route: "keys.createKey",
        prefixUrl: API_UNKEY_DEV_V1,
        defaultValues: {
          apiId: (cache: any) => cache.apiId ?? "",
        },
      },
      "keys.getKey": {
        method: "GET",
        route: "keys.getKey",
        prefixUrl: API_UNKEY_DEV_V1,
        defaultValues: {
          keyId: (cache: any) => cache.keyId ?? "",
        },
      },
      "keys.updateKey": {
        method: "POST",
        route: "keys.updateKey",
        prefixUrl: API_UNKEY_DEV_V1,
        defaultValues: {
          keyId: (cache: any) => cache.id ?? "",
          ownerId: "acme-inc",
          // expires: undefined,
        },
        wontUpdateCache: true,
      },
      "keys.verifyKey": {
        method: "POST",
        route: "keys.verifyKey",
        prefixUrl: API_UNKEY_DEV_V1,
        defaultValues: {
          apiId: process.env.NEXT_PUBLIC_PLAYGROUND_API_ID,
          key: (cache: any) => {
            console.log({ cache });

            return cache.keyId ?? "";
          },
        },
      },
      "keys.getVerifications": {
        method: "GET",
        route: "keys.getVerifications",
        prefixUrl: API_UNKEY_DEV_V1,
        defaultValues: {
          keyId: (cache: any) => cache.keyId ?? "",
        },
        wontUpdateCache: true,
      },
      "keys.deleteKey": {
        method: "GET",
        route: "keys.deleteKey",
        prefixUrl: API_UNKEY_DEV_V1,
        defaultValues: {
          keyId: (cache: any) => cache.keyId ?? "",
        },
        wontUpdateCache: true,
      },
    };
  }, []);

  const STEP_BY_IDX = React.useMemo(() => {
    return [
      { endpoints: [ALL_ENDPOINTS["apis.createApi"]] },
      { endpoints: [ALL_ENDPOINTS["keys.createKey"]] },
      { endpoints: [ALL_ENDPOINTS["keys.getKey"]] },
      { endpoints: [ALL_ENDPOINTS["keys.updateKey"]] },
      { endpoints: [ALL_ENDPOINTS["keys.verifyKey"]] },
      { endpoints: [ALL_ENDPOINTS["keys.updateKey"]] },
      { endpoints: [ALL_ENDPOINTS["keys.verifyKey"]] },
      { endpoints: [ALL_ENDPOINTS["keys.getVerifications"]] },
      { endpoints: [ALL_ENDPOINTS["keys.deleteKey"]] },
      { endpoints: [ALL_ENDPOINTS["keys.verifyKey"]] },
    ];
  }, [ALL_ENDPOINTS]);

  const [endpointTab, setEndpointTab] =
    React.useState<keyof typeof ALL_ENDPOINTS>("apis.createApi");

  async function handleSubmitForm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const formData = new FormData(event.target as HTMLFormElement);
    const variables = Object.fromEntries(formData);

    const endpoint = ALL_ENDPOINTS[endpointTab];
    let url = endpoint.prefixUrl + endpoint.route;

    if ("mockedRequest" in endpoint) {
      willUpdateCacheRef.current = !(
        "wontUpdateCache" in endpoint && (endpoint.wontUpdateCache as boolean)
      );
      setLastResponseJson(endpoint.mockedRequest());
      setStepIdx((prev) => prev + 1);
      setEndpointTab(STEP_BY_IDX[stepIdx + 1]!.endpoints[0].route as any);
      return;
    }

    if (endpoint.method === "GET") {
      const searchParams = new URLSearchParams();
      for (const [key, value] of Object.entries(variables)) {
        searchParams.append(key, value as string);
      }
      url += `?${searchParams.toString()}`;
    }

    const fetchProtectedPayload = protectedApiRequestSchema.parse({
      url,
      method: endpoint.method,
      jsonBody: endpoint.method !== "GET" ? JSON.stringify(variables) : undefined,
    });

    const fetchedProtected = await fetch(`${getBaseUrl()}/api`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(fetchProtectedPayload),
    });

    const jsonText = JSON.stringify(await fetchedProtected.json(), null, 2);
    willUpdateCacheRef.current = !(
      "wontUpdateCache" in endpoint && (endpoint.wontUpdateCache as boolean)
    );
    setLastResponseJson(jsonText);
    setStepIdx((prev) => prev + 1);
    setEndpointTab(STEP_BY_IDX[stepIdx + 1]!.endpoints[0].route as any);
  }

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
        <div className="w-full flex flex-col grow-0 shrink-0 px-5 h-[298px] gap-3.5 overflow-y-scroll">
          {stepIdx > 0 && (
            <div className="w-full flex flex-col">
              <span className="uppercase text-[#A1A1A1]">Last response</span>

              <div className="mt-2.5">
                <Textarea
                  className="resize-none font-mono h-[80px] text-xs p-3 text-[#686868]"
                  value={lastResponseJson ?? '{ "whoops": "nothing to show." }'}
                  readOnly
                />
              </div>
            </div>
          )}

          <div className="flex flex-col">
            <span className="uppercase text-[#A1A1A1]">
              {stepIdx === 0 && "Introduction"}
              {stepIdx !== 0 && "Overview"}
            </span>

            <div className="mt-3.5">
              {stepIdx === 0 && (
                <>
                  Let's get started!
                  <br />
                  <br />
                  Register your API by calling the official Unkey endpoint.
                  <br />
                  <br />
                  Name your API below, then send the request to our endpoint!
                </>
              )}
              {stepIdx === 1 && (
                <>
                  You've successfully registered your API!
                  <br />
                  <br />
                  Let's create your first key!
                </>
              )}
              {stepIdx === 2 && `Now that your key is created, let's fetch information about it.`}
              {stepIdx === 3 && (
                <>
                  As you can see, you just got details such as workspaceId, even roles and
                  permissions.
                  <br />
                  <br />
                  Now, let's assume we want to link the key to a specific user or identifier. We can
                  do that by setting up an ownerId!
                  <br />
                  <br />
                  As an example, you could mark all employes from ACME Company with an ownerId like
                  "acme-inc". That will facilitate searching for all keys used by ACME in the
                  future.
                </>
              )}
            </div>
          </div>
        </div>

        {/* Right panel */}
        <form
          className="w-full pt-2.5 flex flex-col flex-1 px-5 justify-between gap-2 bg-[#080808] border-t-2 border-[#212121]"
          onSubmit={handleSubmitForm}
        >
          <div className="flex flex-col w-full">
            <legend className="uppercase text-[#A1A1A1]">Select Unkey Endpoint:</legend>

            <Tabs
              value={endpointTab}
              onValueChange={setEndpointTab as any}
              defaultValue={endpointTab}
              className="w-full mt-3"
            >
              <TabsList className="w-full font-mono overflow-x-scroll max-w-fit">
                {/* <TabsTrigger value="none">
                  None
                </TabsTrigger> */}
                {STEP_BY_IDX[stepIdx].endpoints.map((endpoint) => (
                  <TabsTrigger key={endpoint.route} value={endpoint.route}>
                    {`/${endpoint.method} ${endpoint.route}`}
                  </TabsTrigger>
                ))}
              </TabsList>
              {/* <TabsContent className="mt-3" value="none">
                You haven't selected any endpoints.
                Please select an endpoint to continue!
              </TabsContent> */}
              {STEP_BY_IDX[stepIdx].endpoints.map((endpoint) => (
                <TabsContent key={endpoint.route} className="mt-3" value={endpoint.route}>
                  <fieldset className="w-full flex flex-col">
                    <legend>You'll call the endpoint with variables:</legend>

                    {Object.entries(endpoint.defaultValues).map(([key, defaultValue]) => {
                      let isNumber = false;

                      let initialValue = "";
                      if (typeof defaultValue === "function") {
                        // Retrieve from cache
                        initialValue = defaultValue(cacheParsed) ?? initialValue;
                      } else if (
                        defaultValue !== null &&
                        defaultValue !== undefined &&
                        typeof defaultValue === "string"
                      ) {
                        initialValue = defaultValue;
                      } else if (
                        defaultValue !== null &&
                        defaultValue !== undefined &&
                        typeof defaultValue === "number"
                      ) {
                        initialValue = String(defaultValue);
                        isNumber = true;
                      }

                      return (
                        <NamedInput
                          key={key}
                          label={key}
                          name={key}
                          type={isNumber ? "number" : "text"}
                          step={isNumber ? "1" : undefined}
                          defaultValue={initialValue}
                          className="mt-2 font-mono"
                        />
                      );
                    })}
                  </fieldset>
                </TabsContent>
              ))}
            </Tabs>
          </div>

          <div className="flex flex-col w-full">
            <Button>Send Request</Button>

            <label className="mt-3">Equivalent CURL request:</label>
            <Textarea
              className="mt-2 resize-none font-mono h-[104px] text-xs p-3 text-[#686868]"
              defaultValue={curlEquivalent}
              readOnly
            />
          </div>
        </form>
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
