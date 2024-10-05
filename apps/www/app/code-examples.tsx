"use client";
import { PrimaryButton, SecondaryButton } from "@/components/button";
import { SectionTitle } from "@/components/section";
import { CurlIcon, ElixirIcon, GoIcon, JavaIcon, LangIconProps, PythonIcon, RustIcon, TSIcon } from "@/components/svg/lang-icons";
import { CodeEditor } from "@/components/ui/code-editor";
import { MeteorLines } from "@/components/ui/meteorLines";
import { cn } from "@/lib/utils";
import * as TabsPrimitive from "@radix-ui/react-tabs";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import type { PrismTheme } from "prism-react-renderer";
import React, { useEffect } from "react";
import { useState } from "react";
const Tabs = TabsPrimitive.Root;

const editorTheme = {
  plain: {
    color: "#F8F8F2",
    backgroundColor: "#282A36",
  },
  styles: [
    {
      types: ["keyword"],
      style: {
        color: "#9D72FF",
      },
    },
    {
      types: ["function"],
      style: {
        color: "#FB3186",
      },
    },
    {
      types: ["string"],
      style: {
        color: "#3CEEAE",
      },
    },
    {
      types: ["string-property"],
      style: {
        color: "#9D72FF",
      },
    },
    {
      types: ["number"],
      style: {
        color: "#FB3186",
      },
    },
    {
      types: ["comment"],
      style: {
        color: "#4D4D4D",
      },
    },
  ],
} satisfies PrismTheme;

const typescriptCodeBlock = `import { verifyKey } from '@unkey/api';

const { result, error } = await verifyKey({
  apiId: "api_123",
  key: "xyz_123"
})

if ( error ) {
  // handle network error
}

if ( !result.valid ) {
  // reject unauthorized request
}

// handle request`;

const nextJsCodeBlock = `import { withUnkey } from '@unkey/nextjs';
export const POST = withUnkey(async (req) => {
  // Process the request here
  // You have access to the typed verification response using \`req.unkey\`
  console.log(req.unkey);
  return new Response('Your API key is valid!');
});`;

const nuxtCodeBlock = `export default defineEventHandler(async (event) => {
  if (!event.context.unkey.valid) {
    throw createError({ statusCode: 403, message: "Invalid API key" })
  }

  // return authorised information
  return {
    // ...
  };
});`;

const pythonCodeBlock = `import asyncio
import os
import unkey

async def main() -> None:
  client = unkey.Client(api_key=os.environ["API_KEY"])
  await client.start()

  result = await client.keys.verify_key("prefix_abc123")

 if result.is_ok:
   print(data.valid)
 else:
   print(result.unwrap_err())`;

const pythonFastAPICodeBlock = `import os
from typing import Any, Dict, Optional

import fastapi  # pip install fastapi
import unkey  # pip install unkey.py
import uvicorn  # pip install uvicorn

app = fastapi.FastAPI()


def key_extractor(*args: Any, **kwargs: Any) -> Optional[str]:
    if isinstance(auth := kwargs.get("authorization"), str):
        return auth.split(" ")[-1]

    return None


@app.get("/protected")
@unkey.protected(os.environ["UNKEY_API_ID"], key_extractor)
async def protected_route(
    *,
    authorization: str = fastapi.Header(None),
    unkey_verification: Any = None,
) -> Dict[str, Optional[str]]:
    assert isinstance(unkey_verification, unkey.ApiKeyVerification)
    assert unkey_verification.valid
    print(unkey_verification.owner_id)
    return {"message": "protected!"}


if __name__ == "__main__":
    uvicorn.run(app)
`;

const honoCodeBlock = `import { Hono } from "hono"
import { UnkeyContext, unkey } from "@unkey/hono";

const app = new Hono<{ Variables: { unkey: UnkeyContext } }>();
app.use("*", unkey());

app.get("/somewhere", (c) => {
  // access the unkey response here to get metadata of the key etc
  const unkey = c.get("unkey")
 return c.text("yo")
})`;

const tsRatelimitCodeBlock = `import { Ratelimit } from "@unkey/ratelimit"

const unkey = new Ratelimit({
  rootKey: process.env.UNKEY_ROOT_KEY,
  namespace: "my-app",
  limit: 10,
  duration: "30s",
  async: true
})

// elsewhere
async function handler(request) {
  const identifier = request.getUserId() // or ip or anything else you want
  
  const ratelimit = await unkey.limit(identifier)
  if (!ratelimit.success){
    return new Response("try again later", { status: 429 })
  }
  
  // handle the request here
  
}`;

const goVerifyKeyCodeBlock = `package main
import (
	"fmt"
	unkey "github.com/WilfredAlmeida/unkey-go/features"
)
func main() {
	apiKey := "key_3ZZ7faUrkfv1YAhffAcnKW74"
	response, err := unkey.KeyVerify(apiKey)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if response.Valid {
		fmt.Println("Key is valid")
	} else {
		fmt.Println("Key is invalid")
	}
}`;

const goCreateKeyCodeBlock = `package main

import (
	"fmt"

	unkey "github.com/WilfredAlmeida/unkey-go/features"
)

func main() {
	// Prepare the request body
	request := unkey.KeyCreateRequest{
		APIId:      "your-api-id",
		Prefix:     "your-prefix",
		ByteLength: 16,
		OwnerId:    "your-owner-id",
		Meta:       map[string]string{"key": "value"},
		Expires:    0,
		Remaining:  0,
		RateLimit: unkey.KeyCreateRateLimit{
			Type:           "fast",
			Limit:          100,
			RefillRate:     10,
			RefillInterval: 60,
		},
	}

	// Provide the authentication token
	authToken := "your-auth-token"

	// Call the KeyCreate function
	response, err := unkey.KeyCreate(request, authToken)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Process the response
	fmt.Println("Key:", response.Key)
	fmt.Println("Key ID:", response.KeyId)
}

`;

const curlVerifyCodeBlock = `curl --request POST \\
  --url https://api.unkey.dev/v1/keys.verifyKey \\
  --header 'Content-Type: application/json' \\
  --data '{
    "apiId": "api_1234",
    "key": "sk_1234",
  }'`;

const curlCreateKeyCodeBlock = `curl --request POST \\
  --url https://api.unkey.dev/v1/keys.createKey \\
  --header 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  --header 'Content-Type: application/json' \\
  --data '{
    "apiId": "api_123",
    "ownerId": "user_123",
    "expires": ${Date.now() + 7 * 24 * 60 * 60 * 1000},
    "ratelimit": {
      "type": "fast",
      "limit": 10,
      "duration": 60_000
    },
  }'`;

const curlRatelimitCodeBlock = `curl --request POST \
  --url https://api.unkey.dev/v1/ratelimits.limit \
  --header 'Authorization: Bearer <token>' \
  --header 'Content-Type: application/json' \
  --data '{
    "namespace": "email.outbound",
    "identifier": "user_123",
    "limit": 10,
    "duration": 60000,
    "async": true
}'`;

const elixirCodeBlock = `UnkeyElixirSdk.verify_key("xyz_AS5HDkXXPot2MMoPHD8jnL")
# returns
%{"valid" => true,
  "ownerId" => "chronark",
  "meta" => %{
    "hello" => "world"
  }}`;

const rustCodeBlock = `use unkey::models::{VerifyKeyRequest, Wrapped};
use unkey::Client;

async fn verify_key() {
    let api_key = env::var("UNKEY_API_KEY").expect("Environment variable UNKEY_API_KEY not found");
    let c = Client::new(&api_key);
    let req = VerifyKeyRequest::new("test_req", "api_458vdYdbwut5LWABzXZP3Z8jPVas");

    match c.verify_key(req).await {
        Wrapped::Ok(res) => println!("{res:?}"),
        Wrapped::Err(err) => eprintln!("{err:?}"),
    }
}`;

const javaVerifyKeyCodeBlock = `package com.example.myapp;
import com.unkey.unkeysdk.dto.KeyVerifyRequest;
import com.unkey.unkeysdk.dto.KeyVerifyResponse;

@RestController
public class APIController {

    private static IKeyService keyService = new KeyService();

    @PostMapping("/verify")
    public KeyVerifyResponse verifyKey(
        @RequestBody KeyVerifyRequest keyVerifyRequest) {
        // Delegate the creation of the key to the KeyService from the SDK
        return keyService.verifyKey(keyVerifyRequest);
    }
}`;

const javaCreateKeyCodeBlock = `package com.example.myapp;

import com.unkey.unkeysdk.dto.KeyCreateResponse;
import com.unkey.unkeysdk.dto.KeyCreateRequest;

@RestController
public class APIController {

    private static IKeyService keyService = new KeyService();

    @PostMapping("/createKey")
    public KeyCreateResponse createKey(
            @RequestBody KeyCreateRequest keyCreateRequest,
            @RequestHeader("Authorization") String authToken) {
        // Delegate the creation of the key to the KeyService from the SDK
        return keyService.createKey(keyCreateRequest, authToken);
    }
}

`;

type Framework = {
  name: string;
  Icon: React.FC<LangIconProps>;
  codeBlock: string;
  editorLanguage: string;
};

const languagesList = {
  Typescript: [
    {
      name: "Typescript",
      Icon: TSIcon,
      codeBlock: typescriptCodeBlock,
      editorLanguage: "ts",
    },
    {
      name: "Next.js",
      Icon: TSIcon,
      codeBlock: nextJsCodeBlock,
      editorLanguage: "ts",
    },
    {
      name: "Nuxt",
      codeBlock: nuxtCodeBlock,
      Icon: TSIcon,
      editorLanguage: "ts",
    },
    {
      name: "Hono",
      Icon: TSIcon,
      codeBlock: honoCodeBlock,
      editorLanguage: "ts",
    },
    {
      name: "Ratelimiting",
      Icon: TSIcon,
      codeBlock: tsRatelimitCodeBlock,
      editorLanguage: "ts",
    },
  ],
  Python: [
    {
      name: "Python",
      Icon: PythonIcon,
      codeBlock: pythonCodeBlock,
      editorLanguage: "python",
    },
    {
      name: "FastAPI",
      Icon: PythonIcon,
      codeBlock: pythonFastAPICodeBlock,
      editorLanguage: "python",
    },
  ],
  Golang: [
    {
      name: "Verify key",
      Icon: GoIcon,
      codeBlock: goVerifyKeyCodeBlock,
      editorLanguage: "go",
    },
    {
      name: "Create key",
      Icon: GoIcon,
      codeBlock: goCreateKeyCodeBlock,
      editorLanguage: "go",
    },
  ],
  Java: [
    {
      name: "Verify key",
      Icon: JavaIcon,
      codeBlock: javaVerifyKeyCodeBlock,
      editorLanguage: "ts",
    },
    {
      name: "Create key",
      Icon: JavaIcon,
      codeBlock: javaCreateKeyCodeBlock,
      editorLanguage: "ts",
    },
  ],
  Elixir: [
    {
      name: "Verify key",
      Icon: ElixirIcon,
      codeBlock: elixirCodeBlock,
      editorLanguage: "ts",
    },
  ],
  Rust: [
    {
      name: "Verify key",
      Icon: RustIcon,
      codeBlock: rustCodeBlock,
      editorLanguage: "rust",
    },
  ],
  Curl: [
    {
      name: "Verify key",
      Icon: CurlIcon,
      codeBlock: curlVerifyCodeBlock,
      editorLanguage: "tsx",
    },
    {
      name: "Create key",
      Icon: CurlIcon,
      codeBlock: curlCreateKeyCodeBlock,
      editorLanguage: "tsx",
    },
    {
      name: "Ratelimit",
      Icon: CurlIcon,
      codeBlock: curlRatelimitCodeBlock,
      editorLanguage: "tsx",
    },
  ],
} as const satisfies {
  [key: string]: Framework[];
};

// const TabsContent = React.forwardRef<
//   React.ElementRef<typeof TabsPrimitive.Content>,
//   React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
// >(({ className, ...props }, ref) => (
//   <TabsPrimitive.Content
//     ref={ref}
//     className={cn(
//       "mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
//       className,
//     )}
//     {...props}
//   />
// ));
// TabsContent.displayName = TabsPrimitive.Content.displayName;

type Props = {
  className?: string;
};
type Language = "Typescript" | "Python" | "Rust" | "Golang" | "Curl" | "Elixir" | "Java";
type LanguagesList = {
  name: Language;
  Icon: React.FC<LangIconProps>;
};
const languages = [
  { name: "Typescript", Icon: TSIcon },
  { name: "Python", Icon: PythonIcon },
  { name: "Rust", Icon: RustIcon },
  { name: "Golang", Icon: GoIcon },
  { name: "Curl", Icon: CurlIcon },
  { name: "Elixir", Icon: ElixirIcon },
  { name: "Java", Icon: JavaIcon },
] as LanguagesList[];

// TODO extract this automatically from our languages array
type FrameworkName = (typeof languagesList)[Language][number]["name"];

const LanguageTrigger = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>
>(({ className, value, ...props }, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    value={value}
    className={cn(
      "inline-flex items-center gap-1 justify-center whitespace-nowrap rounded-t-lg px-3  py-1.5 text-sm transition-all hover:text-white/80 disabled:pointer-events-none disabled:opacity-50 bg-gradient-to-t from-black to-black data-[state=active]:from-white/10 border border-b-0 text-white/30 data-[state=active]:text-white border-[#454545] font-light",
      className,
    )}
    {...props}
  />
));

LanguageTrigger.displayName = TabsPrimitive.Trigger.displayName;

export const CodeExamples: React.FC<Props> = ({ className }) => {
  const [language, setLanguage] = useState<Language>("Typescript");
  const [framework, setFramework] = useState<FrameworkName>("Typescript");
  const [languageHover, setLanguageHover] = useState("Typescript");
  function getLanguage({ language, framework }: { language: Language; framework: FrameworkName }) {
    const frameworks = languagesList[language];
    const currentFramework = frameworks.find((f) => f.name === framework);
    return currentFramework?.editorLanguage || "tsx";
  }

  useEffect(() => {
    setFramework(languagesList[language].at(0)!.name);
  }, [language]);

  function getCodeBlock({ language, framework }: { language: Language; framework: FrameworkName }) {
    const frameworks = languagesList[language];
    const currentFramework = frameworks.find((f) => f.name === framework);
    return currentFramework?.codeBlock || "";
  }

  const [copied, setCopied] = useState(false);
  return (
    <section className={className}>
      <SectionTitle
        label="Code"
        title="Any language, any framework, always secure"
        text="Add authentication to your APIs in a few lines of code. We provide SDKs for a range of languages and frameworks, and an intuitive REST API with public OpenAPI spec."
        align="center"
        className="relative"
      >
        <div className="absolute bottom-32 left-[-50px]">
          <MeteorLines className="ml-2 fade-in-0" delay={3} number={1} />
          <MeteorLines className="ml-10 fade-in-40" delay={0} number={1} />
          <MeteorLines className="ml-16 fade-in-100" delay={5} number={1} />
        </div>
        <div className="absolute bottom-32 right-[200px]">
          <MeteorLines className="ml-2 fade-in-0" delay={4} number={1} />
          <MeteorLines className="ml-10 fade-in-40" delay={0} number={1} />
          <MeteorLines className="ml-16 fade-in-100" delay={2} number={1} />
        </div>
        <div className="mt-10">
          <div className="flex gap-6 pb-14">
            <Link key="get-started" href="https://app.unkey.com">
              <PrimaryButton shiny label="Get Started" IconRight={ChevronRight} />
            </Link>
            <Link key="docs" href="/docs">
              <SecondaryButton label="Visit the docs" IconRight={ChevronRight} />
            </Link>
          </div>
        </div>
      </SectionTitle>
      <div className="relative w-full rounded-4xl border-[.75px] border-white/10 bg-gradient-to-b from-[#111111] to-black border-t-[.75px] border-t-white/20">
        <div
          aria-hidden
          className="absolute pointer-events-none inset-x-16 h-[432px] bottom-[calc(100%-2rem)] bg-[radial-gradient(94.69%_94.69%_at_50%_100%,rgba(255,255,255,0.20)_0%,rgba(255,255,255,0)_55.45%)]"
        />
        <Tabs
          defaultValue={language}
          onValueChange={(l) => setLanguage(l as Language)}
          className="relative flex items-end h-16 px-4 border rounded-tr-3xl rounded-tl-3xl border-white/10 editor-top-gradient"
        >
          <TabsPrimitive.List className="flex items-end gap-4 overflow-x-auto scrollbar-hidden">
            {languages.map(({ name, Icon }) => (
              <LanguageTrigger
                key={name}
                onMouseEnter={() => setLanguageHover(name)}
                onMouseLeave={() => setLanguageHover(language)}
                value={name}
              >
                <Icon active={languageHover === name || language === name} />
                {name}
              </LanguageTrigger>
            ))}
          </TabsPrimitive.List>
        </Tabs>
        <div className="flex flex-col sm:flex-row overflow-x-auto scrollbar-hidden sm:h-[520px]">
          <FrameworkSwitcher
            frameworks={languagesList[language]}
            currentFramework={framework}
            setFramework={setFramework}
          />
          <div className="relative flex w-full pt-4 pb-8 pl-8 font-mono text-xs text-white sm:text-sm">
            <CodeEditor
              language={getLanguage({ language, framework })}
              theme={editorTheme}
              codeBlock={getCodeBlock({ language, framework })}
            />
            <button
              type="button"
              aria-label="Copy code"
              className="absolute hidden cursor-pointer top-5 right-5 lg:flex"
              onClick={() => {
                navigator.clipboard.writeText(getCodeBlock({ language, framework }));
                setCopied(true);
                setTimeout(() => {
                  setCopied(false);
                }, 2000);
              }}
            >
              {copied ? (
                <svg className="checkmark" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 52 52">
                  <circle className="checkmark__circle" cx="26" cy="26" r="25" fill="none" />
                  <path className="checkmark__check" fill="none" d="M14.1 27.2l7.1 7.2 16.7-16.8" />
                </svg>
              ) : (
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  className=""
                >
                  <path
                    opacity="0.3"
                    fillRule="evenodd"
                    clipRule="evenodd"
                    d="M13 5.00002C13.4886 5.00002 13.6599 5.00244 13.7927 5.02884C14.3877 5.1472 14.8528 5.61235 14.9712 6.20738C14.9976 6.34011 15 6.5114 15 7.00002H16L16 6.94215V6.94213C16.0001 6.53333 16.0002 6.25469 15.952 6.01229C15.7547 5.02057 14.9795 4.24532 13.9877 4.04806C13.7453 3.99984 13.4667 3.99991 13.0579 4.00001L13 4.00002H7.70002H7.67861C7.13672 4.00001 6.69965 4.00001 6.34571 4.02893C5.98128 4.0587 5.66119 4.12161 5.36504 4.2725C4.89464 4.51219 4.51219 4.89464 4.2725 5.36504C4.12161 5.66119 4.0587 5.98128 4.02893 6.34571C4.00001 6.69965 4.00001 7.13672 4.00002 7.67862V7.70002V13L4.00001 13.0579C3.99991 13.4667 3.99984 13.7453 4.04806 13.9877C4.24532 14.9795 5.02057 15.7547 6.01229 15.952C6.25469 16.0002 6.53333 16.0001 6.94213 16H6.94215L7.00002 16V15C6.5114 15 6.34011 14.9976 6.20738 14.9712C5.61235 14.8528 5.1472 14.3877 5.02884 13.7927C5.00244 13.6599 5.00002 13.4886 5.00002 13V7.70002C5.00002 7.13172 5.00041 6.73556 5.02561 6.42714C5.05033 6.12455 5.09642 5.95071 5.16351 5.81903C5.30732 5.53679 5.53679 5.30732 5.81903 5.16351C5.95071 5.09642 6.12455 5.05033 6.42714 5.02561C6.73556 5.00041 7.13172 5.00002 7.70002 5.00002H13ZM11.7 8.00002H11.6786C11.1367 8.00001 10.6996 8.00001 10.3457 8.02893C9.98128 8.0587 9.66119 8.12161 9.36504 8.2725C8.89464 8.51219 8.51219 8.89464 8.2725 9.36504C8.12161 9.66119 8.0587 9.98128 8.02893 10.3457C8.00001 10.6996 8.00001 11.1367 8.00002 11.6786V11.7V16.3V16.3214C8.00001 16.8633 8.00001 17.3004 8.02893 17.6543C8.0587 18.0188 8.12161 18.3388 8.2725 18.635C8.51219 19.1054 8.89464 19.4879 9.36504 19.7275C9.66119 19.8784 9.98128 19.9413 10.3457 19.9711C10.6996 20 11.1366 20 11.6785 20H11.6786H11.7H16.3H16.3214H16.3216C16.8634 20 17.3004 20 17.6543 19.9711C18.0188 19.9413 18.3388 19.8784 18.635 19.7275C19.1054 19.4879 19.4879 19.1054 19.7275 18.635C19.8784 18.3388 19.9413 18.0188 19.9711 17.6543C20 17.3004 20 16.8634 20 16.3216V16.3214V16.3V11.7V11.6786V11.6785C20 11.1366 20 10.6996 19.9711 10.3457C19.9413 9.98128 19.8784 9.66119 19.7275 9.36504C19.4879 8.89464 19.1054 8.51219 18.635 8.2725C18.3388 8.12161 18.0188 8.0587 17.6543 8.02893C17.3004 8.00001 16.8633 8.00001 16.3214 8.00002H16.3H11.7ZM9.81903 9.16351C9.95071 9.09642 10.1246 9.05033 10.4271 9.02561C10.7356 9.00041 11.1317 9.00002 11.7 9.00002H16.3C16.8683 9.00002 17.2645 9.00041 17.5729 9.02561C17.8755 9.05033 18.0493 9.09642 18.181 9.16351C18.4632 9.30732 18.6927 9.53679 18.8365 9.81903C18.9036 9.95071 18.9497 10.1246 18.9744 10.4271C18.9996 10.7356 19 11.1317 19 11.7V16.3C19 16.8683 18.9996 17.2645 18.9744 17.5729C18.9497 17.8755 18.9036 18.0493 18.8365 18.181C18.6927 18.4632 18.4632 18.6927 18.181 18.8365C18.0493 18.9036 17.8755 18.9497 17.5729 18.9744C17.2645 18.9996 16.8683 19 16.3 19H11.7C11.1317 19 10.7356 18.9996 10.4271 18.9744C10.1246 18.9497 9.95071 18.9036 9.81903 18.8365C9.53679 18.6927 9.30732 18.4632 9.16351 18.181C9.09642 18.0493 9.05033 17.8755 9.02561 17.5729C9.00041 17.2645 9.00002 16.8683 9.00002 16.3V11.7C9.00002 11.1317 9.00041 10.7356 9.02561 10.4271C9.05033 10.1246 9.09642 9.95071 9.16351 9.81903C9.30732 9.53679 9.53679 9.30732 9.81903 9.16351Z"
                    fill="url(#paint0_linear_840_3800)"
                  />
                  <defs>
                    <linearGradient
                      id="paint0_linear_840_3800"
                      x1="4.15606"
                      y1="2.27462"
                      x2="4.15606"
                      y2="20.9494"
                      gradientUnits="userSpaceOnUse"
                    >
                      <stop stopColor="white" stopOpacity="0.4" />
                      <stop offset="1" stopColor="white" />
                    </linearGradient>
                  </defs>
                </svg>
              )}
            </button>
          </div>
        </div>
      </div>
    </section>
  );
};

function FrameworkSwitcher({
  frameworks,
  currentFramework,
  setFramework,
}: {
  frameworks: Framework[];
  currentFramework: FrameworkName;
  setFramework: React.Dispatch<React.SetStateAction<FrameworkName>>;
}) {
  return (
    <div className="flex flex-col justify-between sm:w-[216px] text-white text-sm pt-6 px-4 font-mono md:border-r md:border-white/10">
      <div className="flex items-center space-x-2 sm:flex-col sm:space-x-0 sm:space-y-2">
        {frameworks.map((framework) => (
          <button
            key={framework.name}
            type="button"
            onClick={() => {
              setFramework(framework.name as FrameworkName);
            }}
            className={cn(
              "flex items-center cursor-pointer hover:bg-white/10 py-1 px-2 rounded-lg w-[184px] ",
              {
                "bg-white/10 text-white": currentFramework === framework.name,
                "text-white/40": currentFramework !== framework.name,
              },
            )}
          >
            <div>{framework.name}</div>
          </button>
        ))}
      </div>
    </div>
  );
}
