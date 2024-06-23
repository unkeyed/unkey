"use client";

import { Loader2 } from "lucide-react";
import Link from "next/link";
import React, { type FormEvent } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { Button } from "@/components/ui/button";
import { NamedInput } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { protectedApiRequestSchema } from "@/lib/schemas";
import { cn } from "@/lib/utils";

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

const formDataSchema = z.record(z.string());

type CacheValue = string | number | null;
type CacheKV = Record<string, CacheValue>;

type EndpointField = {
  getDefaultValue: () => CacheValue;
  placeholder?: string;
  regexp?: string;
  schema: z.ZodType<CacheValue>;
  cacheAs?: string;
  onResponse?: (response: any) => void;
};
interface Endpoint {
  method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
  route: string;
  prefixUrl: string;
  fields: Record<string, EndpointField>;
  mockedRequest?: () => string;
  getMutatedCache?: (cache: CacheKV, payload: any, response: any) => CacheKV;
}

interface Step {
  endpoint: Endpoint;
  onResponse?: (response: any) => void;
  getJSXText: () => React.ReactNode;
}

export default function Page() {
  const [isDone, setIsDone] = React.useState(false);
  const [stepIdx, setStepIdx] = React.useState(0);
  const [lastResponseJson, setLastResponseJson] = React.useState<string | null>(null);
  const cache = React.useRef<CacheKV>({});
  const [loading, setLoading] = React.useState(false);

  const ALL_ENDPOINTS = React.useMemo(() => {
    return {
      "apis.createApi": {
        method: "POST",
        route: "apis.createApi",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          name: {
            getDefaultValue: () => "my-untitled-api",
            schema: z.string().min(3),
          },
        },
        mockedRequest: () => {
          return JSON.stringify({ apiId: process.env.NEXT_PUBLIC_PLAYGROUND_API_ID }, null, 2);
        },
        getMutatedCache: (cache, _, response) => {
          cache.apiId = response.apiId;
          return cache;
        },
      },
      "keys.createKey": {
        method: "POST",
        route: "keys.createKey",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          apiId: {
            getDefaultValue: () => cache.current.apiId ?? "",
            schema: z.string(),
          },
        },
        getMutatedCache: (cache, _, response) => {
          cache.keyId = response.keyId;
          cache.key = response.key;
          return cache;
        },
      },
      "keys.getKey": {
        method: "GET",
        route: "keys.getKey",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          keyId: {
            getDefaultValue: () => cache.current.keyId ?? "",
            schema: z.string(),
          },
        },
      },
      "keys.updateKey": {
        method: "POST",
        route: "keys.updateKey",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          keyId: {
            getDefaultValue: () => cache.current.keyId ?? "",
            schema: z.string(),
          },
          ownerId: {
            getDefaultValue: () => cache.current.ownerId ?? "acme-inc",
            schema: z.string().min(1).nullable(),
            cacheAs: "ownerId",
            regexp: "^[A-Za-z0-9_-]+$",
          },
          expires: {
            getDefaultValue: () => null,
            schema: z.coerce.number().nullable(),
          },
        },
        getMutatedCache: (cache, payload) => {
          cache.ownerId = payload.ownerId;
          return cache;
        },
      },
      "keys.verifyKey": {
        method: "POST",
        route: "keys.verifyKey",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          apiId: { getDefaultValue: () => cache.current.apiId, schema: z.string() },
          key: { getDefaultValue: () => cache.current.key ?? "", schema: z.string() },
        },
      },
      "keys.getVerifications": {
        method: "GET",
        route: "keys.getVerifications",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          keyId: { getDefaultValue: () => cache.current.keyId ?? "", schema: z.string() },
        },
      },
      "keys.deleteKey": {
        method: "POST",
        route: "keys.deleteKey",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          keyId: { getDefaultValue: () => cache.current.keyId ?? "", schema: z.string() },
        },
      },
    } satisfies { [key: string]: Endpoint };
  }, []);

  const STEP_BY_IDX = React.useMemo<Step[]>(() => {
    return [
      {
        endpoint: ALL_ENDPOINTS["apis.createApi"],
        onResponse: () => {
          toast("You just created an API! 🎉", {});
        },
        getJSXText: () => {
          return (
            <>
              <strong>Welcome to the Unkey playground! 👋</strong>
              <br />
              <br />
              To get started, create an API by calling the official Unkey endpoint below.
              <br />
              <br />
              We've auto-filled most variables for you, such as the API's <Code>name</Code>. Feel
              free to change whatever you like to!
              <br />
              <br />
              Press the white button below to create your API!
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.createKey"],
        onResponse: () => {
          toast("You created your first key! 🔑");
        },
        getJSXText: () => {
          return (
            <>
              You've successfully registered your API!
              <br />
              <strong>Let's create your first key</strong> using that <Code>apiId</Code>.
              <br />
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.verifyKey"],
        onResponse: () => {
          toast("Congrats on your first API key verification!", {
            description: `Each verification will add analytical data we'll get in a later step.`,
          });
        },
        getJSXText: () => {
          return (
            <>
              Now, you have created a key for your API.
              <br />
              Please notice the <Code>keyId</Code> and <Code>key</Code> are not the same.
              <br />
              <br />
              Let's take the <Code>key</Code> together with your <Code>apiId</Code> to consume it
              for the first time.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.getKey"],
        onResponse: () => {
          toast("You retrieved information about a key.");
        },
        getJSXText: () => {
          return (
            <>
              You just verified an API key.
              <br />
              That means the key has been used at least once. Let's fetch more information about it.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.updateKey"],
        onResponse: () => {
          toast("You updated the key by setting an ownerId! ⚒️", {});
        },
        getJSXText: () => {
          return (
            <>
              You just fetched information regarding your recently created API key, such as{" "}
              <Code>workspaceId</Code>, <Code>roles</Code> and <Code>permissions</Code>.
              <br />
              <br />
              Now, let's assume we want to link the key to a specific user or identifier. We can do
              that by setting up an <Code>ownerId</Code>.
              <br />
              <br />
              As an example, you could mark all employees from ACME company with an{" "}
              <Code>ownerId</Code> equal to <Code>acme-inc</Code>. That will facilitate filtering
              key usage by ACME at any point in the future.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.verifyKey"],
        onResponse: () => {
          toast("You just verified the key.");
        },
        getJSXText: () => {
          return <>Lorem ipsum dolor</>;
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.updateKey"],
        onResponse: () => {
          toast("You just updated the key to add an expiration date! ⚒️", {});
        },
        getJSXText: () => {
          return <>Lorem ipsum dolor</>;
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.verifyKey"],
        onResponse: () => {
          toast("You just verified the key.", {});
        },
        getJSXText: () => {
          return <>Lorem ipsum dolor</>;
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.getVerifications"],
        onResponse: () => {
          toast("You retrieved analytical data! 🔍", {});
        },
        getJSXText: () => {
          return <>Lorem ipsum dolor</>;
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.deleteKey"],
        onResponse: () => {
          toast("You deleted your first key! 🗑️", {});
        },
        getJSXText: () => {
          return <>Lorem ipsum dolor</>;
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.verifyKey"],
        onResponse: () => {
          toast("Congratulations! 🎉", {
            action: {
              label: "Try Unkey",
              onClick: () => window.open("https://app.unkey.com/auth/sign-up"),
            },
            // action: <ToastAction altText="Sign up">Sign up</ToastAction>,
          });
        },
        getJSXText: () => {
          return <>Lorem ipsum dolor</>;
        },
      },
    ] satisfies Step[];
  }, [ALL_ENDPOINTS]);

  const step = STEP_BY_IDX[stepIdx];
  const previousStep = STEP_BY_IDX[stepIdx - 1];

  const getCurlEquivalent = React.useCallback(() => {
    let curlEquivalent = null;
    const form = document.querySelector("#endpoint-form");
    if (step?.endpoint && form instanceof HTMLFormElement) {
      const fd = new FormData(form);
      const formObj = Object.fromEntries(fd);

      curlEquivalent = "";
      curlEquivalent += `curl --request ${step.endpoint.method} `;

      let url = `${step.endpoint.prefixUrl}/${step.endpoint.route}`;
      if (step.endpoint.method === "GET") {
        const searchParams = new URLSearchParams();
        for (const [key, value] of Object.entries(formObj)) {
          searchParams.append(key, value as string);
        }
        url += `?${searchParams.toString()}`;
      }
      curlEquivalent += `\n   --url ${url}`;
      curlEquivalent += `\n   --header 'Authorization: Bearer <token>' `;
      curlEquivalent += `\n   --header 'Content-Type: application/json' `;

      if (step.endpoint.method !== "GET") {
        // TODO: parse obj with zod before converting to json
        const json = JSON.stringify(formObj);
        curlEquivalent += `\n   --data '${json}'`;
      }
    }

    return curlEquivalent;
  }, [step]);

  const [curlEquivalent, setCurlEquivalent] = React.useState<string | null>(getCurlEquivalent());
  React.useEffect(() => {
    setCurlEquivalent(getCurlEquivalent());

    const interval = setInterval(() => {
      setCurlEquivalent(getCurlEquivalent());
    }, 2000);

    return () => {
      clearInterval(interval);
    };
  }, [getCurlEquivalent]);

  async function handleSubmitForm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!step?.endpoint) {
      return;
    }

    /**
     * Parse form data
     */
    const formData = new FormData(event.target as HTMLFormElement);
    const formObj: Record<string, string | null> = formDataSchema.parse(
      Object.fromEntries(formData),
    );
    Object.entries(formObj).forEach(([key, value]) => {
      if (value === "") {
        formObj[key] = null;
      }
    });

    /** Validate form data with Zod or toast an error */
    const zodShape: z.ZodRawShape = {};
    Object.entries(step.endpoint.fields).forEach(([key, value]) => {
      zodShape[key] = (value as EndpointField).schema;
    });
    const payloadSchema = z.object(zodShape);
    const parsedPayload = payloadSchema.safeParse(formObj);
    if (parsedPayload.error && !parsedPayload.success) {
      toast(`Incorrect variable "${parsedPayload.error.errors[0].path}"`, {
        description: parsedPayload.error.errors[0].message,
      });
      return;
    }

    if ("mockedRequest" in step.endpoint) {
      setLoading(true);
      const jsonResponse = step.endpoint.mockedRequest!();
      // Fake delay
      await new Promise((resolve) => setTimeout(resolve, 1000));
      const parsedResponse = JSON.parse(jsonResponse);
      if ("getMutatedCache" in step.endpoint) {
        cache.current = step.endpoint.getMutatedCache!(
          cache.current,
          parsedPayload.data,
          parsedResponse,
        );
      }

      step.onResponse?.(parsedResponse);

      setLastResponseJson(jsonResponse);
      if (stepIdx + 1 === STEP_BY_IDX.length) {
        setIsDone(true);
      } else {
        setStepIdx((prev) => prev + 1);
      }
      setLoading(false);
    }
    if (!("mockedRequest" in step.endpoint)) {
      setLoading(true);
      let url = step.endpoint.prefixUrl + step.endpoint.route;

      if (step.endpoint.method === "GET") {
        const searchParams = new URLSearchParams();
        for (const [key, value] of Object.entries(formObj)) {
          searchParams.append(key, value as string);
        }
        url += `?${searchParams.toString()}`;
      }

      const payload = protectedApiRequestSchema.parse({
        url,
        method: step.endpoint.method,
        jsonBody: step.endpoint.method !== "GET" ? JSON.stringify(formObj) : undefined,
      });

      const response = await fetch(`${getBaseUrl()}/api`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });
      const parsedResponse = await response.json();

      step.onResponse?.(parsedResponse);

      if ("getMutatedCache" in step.endpoint) {
        cache.current = step.endpoint.getMutatedCache!(
          cache.current,
          parsedPayload.data,
          parsedResponse,
        );
      }

      const jsonText = JSON.stringify(parsedResponse, null, 2);

      setLastResponseJson(jsonText);
      if (stepIdx + 1 === STEP_BY_IDX.length) {
        setIsDone(true);
      } else {
        setStepIdx((prev) => prev + 1);
      }
      setLoading(false);
    }
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

      <div
        className={cn(
          "w-full flex flex-col items-center flex-1 h-full max-w-[500px] pb-5 mx-auto",
          isDone && "hidden",
        )}
      >
        {/* Left panel */}
        <div className="w-full flex flex-col grow-0 shrink-0 px-5 h-[298px] gap-3.5 overflow-y-scroll pb-6 leading-[1.7]">
          {stepIdx > 0 && (
            <div className="w-full flex flex-col">
              <span className="uppercase text-[#A1A1A1]">Last response</span>

              <div className="mt-2.5">
                <code
                  className="block bg-transparent rounded-md border border-input px-3 py-2 text-sm shadow-sm resize-none font-mono min-h-max text-xs p-3 text-[#686868] overflow-scroll"
                  dangerouslySetInnerHTML={{
                    __html:
                      (previousStep !== undefined
                        ? `/${previousStep.endpoint.method}  ${previousStep.endpoint.route}<br/>`
                        : "") +
                      (lastResponseJson !== undefined && lastResponseJson !== null
                        ? lastResponseJson
                            .replace(/(?:\r\n|\r|\n)/g, "<br>")
                            .replace(/ /g, "&nbsp;")
                        : '{ "whoops": "nothing to show." }'),
                  }}
                />
              </div>
            </div>
          )}

          <div className="flex flex-col">
            {stepIdx === 0 && <span className="uppercase text-[#A1A1A1]">Introduction</span>}

            {step !== undefined && (
              <div className={cn(stepIdx === 0 && "mt-3.5")}>{step.getJSXText()}</div>
            )}
          </div>
        </div>

        {/* Right panel */}
        <form
          id="endpoint-form"
          className="w-full pt-2.5 flex flex-col flex-1 px-5 justify-between gap-4 bg-[#080808] border-t-2 border-[#212121] shadow-sm [--tw-shadow:0_-10px_14px_0_rgb(0_0_0_/_76%)]"
          onSubmit={handleSubmitForm}
        >
          <div className="flex flex-col w-full">
            <legend className="uppercase text-[#A1A1A1]">
              Call Unkey endpoint with variables:
            </legend>

            {step.endpoint && (
              <div key={step.endpoint.route} className="mt-3">
                <fieldset className="w-full flex flex-col gap-2">
                  {/* <legend>You'll call the endpoint with variables:</legend> */}

                  {Object.entries(step.endpoint.fields).map(([key, _value]) => {
                    const value = _value as EndpointField;

                    const defaultValue = value.getDefaultValue();

                    const formattedDefaultValue = defaultValue === null ? "" : String(defaultValue);

                    const isAutoFilled = cache.current[value.cacheAs ?? key] === defaultValue;

                    return (
                      <div className="relative w-full">
                        <NamedInput
                          key={key}
                          label={key}
                          name={key}
                          type={typeof defaultValue === "number" ? "number" : "text"}
                          step={typeof defaultValue === "number" ? "1" : undefined}
                          defaultValue={formattedDefaultValue}
                          placeholder={value.placeholder}
                          pattern={value.regexp}
                          className="peer font-mono [&_label]:w-[5rem]"
                          readOnly={isAutoFilled}
                        />

                        {isAutoFilled && (
                          <span className="absolute top-0 right-1 text-[#A1A1A1] text-[11px] bg-background px-2 -translate-y-1/2">
                            Smart-filled (readonly)
                          </span>
                        )}
                      </div>
                    );
                  })}
                </fieldset>
              </div>
            )}
          </div>

          {step?.endpoint !== undefined && (
            <div className="flex flex-col w-full font-mono">
              <Button disabled={loading}>
                {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {`/${step.endpoint.method} ${step.endpoint.route}`}
              </Button>

              {curlEquivalent !== null && (
                <>
                  <label className="mt-3">Equivalent CURL request:</label>
                  <Textarea
                    className="mt-2 resize-none font-mono h-[104px] text-xs p-3 text-[#686868]"
                    value={curlEquivalent}
                    readOnly
                  />
                </>
              )}
            </div>
          )}
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

const Code = React.forwardRef<HTMLSpanElement, React.HTMLProps<HTMLElement>>(
  ({ className, ...props }, ref) => {
    return (
      <code
        ref={ref}
        className={cn("rounded bg-muted px-[.3rem] py-[.2rem] font-mono", className)}
        {...props}
      />
    );
  },
);
