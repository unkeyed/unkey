"use client";
import { PrimaryButton, SecondaryButton } from "@/components/button";
import { SectionTitle } from "@/components/section";
import type { LangIconProps } from "@/components/svg/lang-icons";
import {
  CurlIcon,
  ElixirIcon,
  GoIcon,
  JavaIcon,
  PythonIcon,
  RustIcon,
  TSIcon,
} from "@/components/svg/lang-icons";
import { CodeEditor } from "@/components/ui/code-editor";
import { CopyCodeSnippetButton } from "@/components/ui/copy-code-button";
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

import(
	unkeygo "github.com/unkeyed/unkey-go"
	"context"
	"github.com/unkeyed/unkey-go/models/components"
	"log"
)

func main() {
    s := unkeygo.New(
        unkeygo.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    ctx := context.Background()
    res, err := s.Keys.VerifyKey(ctx, components.V1KeysVerifyKeyRequest{
        APIID: unkeygo.String("api_1234"),
        Key: "sk_1234",
        Ratelimits: []components.Ratelimits{
            components.Ratelimits{
                Name: "tokens",
                Limit: unkeygo.Int64(500),
                Duration: unkeygo.Int64(3600000),
            },
            components.Ratelimits{
                Name: "tokens",
                Limit: unkeygo.Int64(20000),
                Duration: unkeygo.Int64(86400000),
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.V1KeysVerifyKeyResponse != nil {
        // handle response
    }
}`;

const goCreateKeyCodeBlock = `package main

import(
	unkeygo "github.com/unkeyed/unkey-go"
	"context"
	"github.com/unkeyed/unkey-go/models/operations"
	"log"
)

func main() {
    s := unkeygo.New(
        unkeygo.WithSecurity("<YOUR_BEARER_TOKEN_HERE>"),
    )

    ctx := context.Background()
    res, err := s.Keys.CreateKey(ctx, operations.CreateKeyRequestBody{
        APIID: "api_123",
        Name: unkeygo.String("my key"),
        ExternalID: unkeygo.String("team_123"),
        Meta: map[string]any{
            "billingTier": "PRO",
            "trialEnds": "2023-06-16T17:16:37.161Z",
        },
        Roles: []string{
            "admin",
            "finance",
        },
        Permissions: []string{
            "domains.create_record",
            "say_hello",
        },
        Expires: unkeygo.Int64(1623869797161),
        Remaining: unkeygo.Int64(1000),
        Refill: &operations.Refill{
            Interval: operations.IntervalDaily,
            Amount: 100,
        },
        Ratelimit: &operations.Ratelimit{
            Type: operations.TypeFast.ToPointer(),
            Limit: 10,
            Duration: unkeygo.Int64(60000),
        },
        Enabled: unkeygo.Bool(false),
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.Object != nil {
        // handle response
    }
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
      editorLanguage: "tsx",
    },
    {
      name: "Next.js",
      Icon: TSIcon,
      codeBlock: nextJsCodeBlock,
      editorLanguage: "tsx",
    },
    {
      name: "Nuxt",
      codeBlock: nuxtCodeBlock,
      Icon: TSIcon,
      editorLanguage: "tsx",
    },
    {
      name: "Hono",
      Icon: TSIcon,
      codeBlock: honoCodeBlock,
      editorLanguage: "tsx",
    },
    {
      name: "Ratelimiting",
      Icon: TSIcon,
      codeBlock: tsRatelimitCodeBlock,
      editorLanguage: "tsx",
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
      editorLanguage: "tsx",
    },
    {
      name: "Create key",
      Icon: JavaIcon,
      codeBlock: javaCreateKeyCodeBlock,
      editorLanguage: "tsx",
    },
  ],
  Elixir: [
    {
      name: "Verify key",
      Icon: ElixirIcon,
      codeBlock: elixirCodeBlock,
      editorLanguage: "tsx",
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
            <CopyCodeSnippetButton
              textToCopy={getCodeBlock({ language, framework })}
              className="absolute hidden cursor-pointer top-5 right-5 lg:flex"
            />
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
