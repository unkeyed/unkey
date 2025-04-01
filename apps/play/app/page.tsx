"use client";

import { Loader2 } from "lucide-react";
import Link from "next/link";
import React, { type FormEvent } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { Button } from "@/components/ui/button";
import { NamedInput } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { protectedApiRequestSchema } from "@/lib/schemas";
import { cn } from "@/lib/utils";
import ms from "ms";

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

// Avoid initial layout shift
const CURL_PLACEHOLDER = `curl --request POST \\\n   --url https://api.unkey.dev/v1/apis.createApi \\\n   --header 'Authorization: Bearer <token>' \\\n   --header 'Content-Type: application/json'  \\\n   --data '{\n     "name": "my-untitled-api"\n   }'`;

const formDataSchema = z.record(z.string());

type CacheValue = string | number | null | undefined;
type CacheKV = Record<string, CacheValue>;

type EndpointField = {
  getDefaultValue: () => CacheValue;
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
            schema: z.string().min(3).max(30),
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
          // TODO: should be optional
          name: {
            getDefaultValue: () => "my-first-key",
            schema: z.string().min(0).max(30),
          },
          // TODO: should be optional
          prefix: {
            getDefaultValue: () => "play",
            schema: z.string().min(0).max(30),
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
      "keys.updateKeyWithOwnerId": {
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
            schema: z
              .string()
              .min(1)
              .max(30)
              .regex(/^[A-Za-z0-9-_]+$/g, "Must be A-Z, a-z, 0-9, - or _"),
            cacheAs: "ownerId",
          },
        },
        getMutatedCache: (cache, payload) => {
          cache.ownerId = payload.ownerId;
          return cache;
        },
      },
      "keys.updateKeyWithExpires": {
        method: "POST",
        route: "keys.updateKey",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          keyId: {
            getDefaultValue: () => cache.current.keyId ?? "",
            schema: z.string(),
          },
          // TODO: is an optional number but should not require making every other field "null" to make it optional
          expires: {
            getDefaultValue: () => Date.now() + 1_000 * 60 * 60,
            schema: z.coerce
              .number()
              .refine((time) => time % 1 === 0, {
                message: "Expiration date should be a valid unix timestamp",
              })
              .refine((time) => time < Date.now() + 1_000 * 60 * 60 * 24 * 365 * 30, {
                message: "Expiration date should be less than 30 years in the future",
              })
              .refine((time) => time > Date.now(), {
                message: "Expiration date should be in the future",
              }),
          },
        },
        getMutatedCache: (cache, payload) => {
          cache.expires = payload.expires;
          return cache;
        },
      },
      "keys.verifyKey": {
        method: "POST",
        route: "keys.verifyKey",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          apiId: {
            getDefaultValue: () => cache.current.apiId,
            schema: z.string(),
          },
          key: {
            getDefaultValue: () => cache.current.key ?? "",
            schema: z.string(),
          },
        },
      },
      "keys.getVerifications": {
        method: "GET",
        route: "keys.getVerifications",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          keyId: {
            getDefaultValue: () => cache.current.keyId ?? "",
            schema: z.string(),
          },
        },
      },
      "keys.deleteKey": {
        method: "POST",
        route: "keys.deleteKey",
        prefixUrl: API_UNKEY_DEV_V1,
        fields: {
          keyId: {
            getDefaultValue: () => cache.current.keyId ?? "",
            schema: z.string(),
          },
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
              To get started, create an API by calling the endpoint.
              <br />
              An API is like a project that contains all your keys and usage data. You can create as
              many APIs for different environments as you like.
              <br />
              <br />
              We've auto-filled the API's <Code>name</Code>. Feel free to change it to whatever you
              like to!
              <br />
              <br />
              Click the button to create your API!
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
              <br />
              <strong>Let's create your first key</strong>.
              <br />
              Creating keys requires the <Code>apiId</Code>. You can configure a lot of settings for
              each key, but let's keep it simple for now.
              <br />
              <br />
              If you want, you can set a <Code>name</Code> for this key to make it easier to
              identify later. As well as a <Code>prefix</Code> that allows your users to identify
              where the key is coming from.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.verifyKey"],
        onResponse: () => {
          toast("Congrats on your first API key verification!", {
            description: `Each verification will add usage data we'll get in a later step.`,
          });
        },
        getJSXText: () => {
          return (
            <>
              Now, you have created a key for your API.
              <br />
              <br />- <Code>keyId</Code> is a unique identifier for the key, you can use it later to
              fetch the key or update it.
              <br />- <Code>key</Code> is the actual secret key.
              <br />
              <br />
              When a user wants to access your API, they will need to provide the <Code>key</Code>{" "}
              in their request.
              <br />
              Let's take the <Code>key</Code> together with your <Code>apiId</Code> to verify it for
              the first time.
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
              <br />
              As you can see, the key has a <Code>valid</Code> field, which indicates if the key is
              valid or not. This is all you need to grant access to your API.
              <br />
              <br />
              There are many more settings you can configure for each key, but first let's fetch the
              key's current settings.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.updateKeyWithOwnerId"],
        onResponse: () => {
          toast("You updated the key by setting an ownerId! ⚒️", {});
        },
        getJSXText: () => {
          return (
            <>
              Great, you've successfully fetched the key! This includes all of the currently
              configured settings for the key.
              <br />
              <br />
              Now, let's assume we want to link the key to a specific user or identifier. We can do
              that by updating the key to include an <Code>ownerId</Code>.
              <br />
              <br />
              As an example, you could mark all employees from ACME company with an{" "}
              <Code>ownerId</Code> equal to <Code>acme-inc</Code>. That will allow you to filter key
              usage by ACME at any point in the future to understand the overall usage of a
              particular customer.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.verifyKey"],
        onResponse: () => {
          toast("You just verified the key.", {
            description: "There's indeed a new ownerId associated!",
          });
        },
        getJSXText: () => {
          return (
            <>
              You just updated the key by setting an <Code>ownerId</Code>.
              <br />
              <br />
              Let's double check the <Code>ownerId</Code> was applied by verifying the key again.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.updateKeyWithExpires"],
        onResponse: () => {
          toast("You just set up an expiration date for that key! ⚒️", {});
        },
        getJSXText: () => {
          return (
            <>
              Well done! Whoever consumes that API key will now be linked to{" "}
              <Code>{cache.current.ownerId}</Code>.
              <br />
              <br />
              Next, let's add a <strong>1-hour expiration time</strong> for this key, using a unix
              timestamp. After the key expires, it will no longer be valid, but you can always
              update it again to extend its life.
              <br />
              Set an <Code>expires</Code> value and update the key again.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.verifyKey"],
        onResponse: () => {
          toast("You just verified the key.", {});
        },
        getJSXText: () => {
          return (
            <>
              You've successfully set up an expiration date for the key through the{" "}
              <Code>expires</Code> variable.
              <br />
              <br />
              Again, let's double check if the <Code>expires</Code> is applied by verifying the key
              again.
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.getVerifications"],
        onResponse: () => {
          toast("You retrieved usage data! 🔍", {});
        },
        getJSXText: () => {
          const remainingExpirationTime = cache.current.expires
            ? ms(new Date(cache.current.expires).getTime() - Date.now())
            : "never";

          return (
            <>
              Seems like the <Code>expires</Code> date is <strong>{remainingExpirationTime}</strong>{" "}
              from now!
              <br />
              We've now used our key more than once. Let's check its usage numbers!
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.deleteKey"],
        onResponse: () => {
          toast("You deleted your first key! 🗑️", {});
        },
        getJSXText: () => {
          return (
            <>
              The total usage numbers reveal that we've used the key three times!
              <br />
              As the API response suggests, we offer a variety of features, such as{" "}
              <Code>per key rate limiting</Code> and <Code>usage based limits</Code>.
              <br />
              <strong>Finally, let's delete our key.</strong>
            </>
          );
        },
      },
      {
        endpoint: ALL_ENDPOINTS["keys.verifyKey"],
        onResponse: () => {
          toast("Congratulations! 🎉", {
            description: "Time to explore our SDK, rate limiting, rich analytics and more!",
            action: {
              label: "Try Unkey",
              onClick: () => window.open("https://go.unkey.com/swag"),
            },
          });
        },
        getJSXText: () => {
          return (
            <>
              You just deleted the key <Code>{cache.current.key}</Code>!
              <br />
              Let's double-check it no longer exists and is invalid from now on.
            </>
          );
        },
      },
    ] satisfies Step[];
  }, [ALL_ENDPOINTS]);

  const step = STEP_BY_IDX[stepIdx];
  const previousStep = STEP_BY_IDX[stepIdx - 1];

  const getCurlEquivalent = React.useCallback(() => {
    let curlEquivalent = CURL_PLACEHOLDER;
    if (typeof window === "undefined" || typeof document === "undefined") {
      return curlEquivalent;
    }

    const form = document.querySelector("#endpoint-form");
    if (step?.endpoint && form instanceof HTMLFormElement) {
      const fd = new FormData(form);
      const formObj = Object.fromEntries(fd);

      curlEquivalent = "";
      curlEquivalent += `curl --request ${step.endpoint.method}`;

      let url = `${step.endpoint.prefixUrl}${step.endpoint.route}`;
      if (step.endpoint.method === "GET") {
        const searchParams = new URLSearchParams();
        for (const [key, value] of Object.entries(formObj)) {
          searchParams.append(key, value as string);
        }
        url += `?${searchParams.toString()}`;
      }
      curlEquivalent += ` \\\n   --url ${url}`;
      curlEquivalent += ` \\\n   --header 'Authorization: Bearer <token>'`;

      if (step.endpoint.method !== "GET") {
        curlEquivalent += ` \\\n   --header 'Content-Type: application/json' `;

        // TODO: parse obj with zod before converting to json
        const json = JSON.stringify(formObj, null, 2)
          .split("\n")
          .map((line, i) => (i === 0 ? line : `   ${line}`))
          .join("\n");
        curlEquivalent += ` \\\n   --data '${json}'`;
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
        jsonBody: step.endpoint.method !== "GET" ? JSON.stringify(parsedPayload.data) : undefined,
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
    <div className="w-full h-full min-h-[100dvh] text-sm lg:text-base text-[#E2E2E2] flex flex-col">
      <header className="w-full flex grow-0 shrink-0 justify-center items-center lg:border-b h-14 text-[#fff] text-xl">
        <nav className="w-full max-w-[1200px] h-full px-5 flex items-center justify-between">
          <div className="flex items-center gap-3 text">
            <Link href="https://unkey.com" target="_blank">
              <h1>
                <SVGLogoUnkey />
              </h1>
            </Link>
            <span>/</span>
            <Link href="https://play.unkey.com" target="_blank">
              <h1>Playground</h1>
            </Link>
          </div>

          <Button asChild className="py-0 h-[28px] px-2">
            <Link href="https://app.unkey.com/auth/sign-up">Try Unkey</Link>
          </Button>
        </nav>
      </header>

      <div
        className={cn(
          "w-full flex flex-col lg:flex-row lg:gap-4 items-center lg:justify-center flex-1 h-full max-w-[1200px] mx-auto lg:px-5",
        )}
      >
        {/* Left panel */}
        <div
          className={cn(
            "w-full lg:w-1/2 flex flex-col grow-0 lg:grow-[unset] shrink-0 lg:shrink-[unset] px-5 gap-3.5 overflow-y-scroll pb-6 lg:p-6 leading-[1.7]",
            !isDone && "h-[298px] lg:h-[500px]",
            "lg:border lg:rounded-xl",
          )}
        >
          {stepIdx > 0 && (
            <div className="flex flex-col w-full">
              <span className="uppercase text-[#A1A1A1]">Last response</span>

              <div className="mt-2.5">
                <code
                  className="block bg-transparent rounded-md border border-input px-3 py-2 shadow-sm resize-none font-mono min-h-max text-xs lg:text-sm p-3 text-[#686868] overflow-scroll"
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
            {stepIdx === 0 && (
              <span className="uppercase text-[#A1A1A1] lg:text-sm">Introduction</span>
            )}

            {!isDone && step !== undefined && (
              <div className={cn(stepIdx === 0 && "mt-3.5")}>{step.getJSXText()}</div>
            )}
            {isDone && (
              <div className={cn(stepIdx === 0 && "mt-3.5")}>
                <strong>Congratulations! 🥳</strong>
                <br />
                You are now ready to enjoy Unkey and explore all of our features.
                <br />
                We offer{" "}
                <Link
                  className="underline"
                  href="https://www.unkey.com/docs/libraries/ts/ratelimit#unkeyratelimit"
                  target="_blank"
                >
                  rate limiting at the edge
                </Link>
                ,{" "}
                <Link
                  className="underline"
                  href="https://www.unkey.com/docs/libraries/ts/ratelimit#unkeyratelimit"
                  target="_blank"
                >
                  custom library integrations
                </Link>
                ,{" "}
                <Link
                  className="underline"
                  href="https://www.unkey.com/docs/libraries/ts/sdk/overview"
                  target="_blank"
                >
                  a TypeScript SDK
                </Link>
                ,{" "}
                <Link
                  className="underline"
                  href="https://www.unkey.com/docs/libraries/go/overview"
                  target="_blank"
                >
                  a Golang SDK
                </Link>
                ,{" "}
                <Link className="underline" href="https://www.unkey.com/pricing" target="_blank">
                  a free tier to boost your start
                </Link>{" "}
                and{" "}
                <Link
                  className="underline"
                  href="https://www.unkey.com/docs/introduction"
                  target="_blank"
                >
                  more
                </Link>
                .
                <br />
                <br />
                The first 200 users to sign up will receive an Unkey swag pack!
                <br />
                <br />
                <Button asChild className="w-full text-center">
                  <Link href="https://go.unkey.com/swag">Get Started for free</Link>
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* Right panel */}
        <form
          id="endpoint-form"
          className={cn(
            "w-full p-5 pt-2.5 lg:p-6 flex flex-col flex-1 justify-between gap-4 bg-[#080808] border-t-2 lg:border-t-0 border-[#212121] shadow-sm lg:shadow-none [--tw-shadow:0_-10px_14px_0_rgb(0_0_0_/_76%)]",
            isDone && "hidden",
            "lg:border lg:rounded-xl lg:h-[500px]",
          )}
          onSubmit={handleSubmitForm}
        >
          <div className="flex flex-col w-full">
            <legend className="uppercase text-[#A1A1A1] lg:text-sm">
              Call Unkey endpoint with variables:
            </legend>

            {step.endpoint && (
              <div key={step.endpoint.route} className="mt-3">
                <fieldset className="flex flex-col w-full gap-2">
                  {/* <legend>You'll call the endpoint with variables:</legend> */}

                  {Object.entries(step.endpoint.fields).map(([key, _value]) => {
                    const value = _value as EndpointField;

                    const defaultValue = value.getDefaultValue();

                    const formattedDefaultValue = defaultValue === null ? "" : String(defaultValue);

                    const isAutoFilled = cache.current[value.cacheAs ?? key] === defaultValue;

                    return (
                      <div key={key} className="relative w-full">
                        <NamedInput
                          key={key}
                          label={key}
                          name={key}
                          type={typeof defaultValue === "number" ? "number" : "text"}
                          step={typeof defaultValue === "number" ? "1" : undefined}
                          defaultValue={formattedDefaultValue}
                          className={cn(
                            "peer font-mono [&_label]:w-[5rem]",
                            isAutoFilled && "opacity-60",
                          )}
                          readOnly={isAutoFilled}
                        />

                        {isAutoFilled && (
                          <span className="absolute top-0 right-1 text-[#A1A1A1] text-[11px] lg:text-[13px] bg-background px-2 -translate-y-1/2">
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
              <Button disabled={loading} variant="default">
                {loading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                {`/${step.endpoint.method} ${step.endpoint.route}`}
              </Button>

              {curlEquivalent !== null && (
                <>
                  <label htmlFor="curl" className="mt-3">
                    Equivalent CURL request:
                  </label>
                  <Textarea
                    id="curl"
                    className="mt-2 resize-none font-mono h-[104px] lg:h-[160px] text-xs lg:text-sm p-3 text-[#686868]"
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

// const Heading = React.forwardRef<HTMLDivElement, React.HTMLProps<HTMLElement>>(
//   ({ children, className, ...props }, ref) => {
//     return (
//       <strong ref={ref} className={cn("text-lg lg:text-xl block mb-1", className)} {...props}>
//         {children}
//         <br />
//       </strong>
//     );
//   },
// );
