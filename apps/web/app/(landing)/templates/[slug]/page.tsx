import { Container } from "@/components/landing/container";
import { PageIntro } from "@/components/landing/page-intro";
import { ExternalLink } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox"
import { languages, templates, frameworks } from "../data";
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'


import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion"
import Link from "next/link";
import { notFound } from "next/navigation";
export const metadata = {
    title: "Templates | Unkey",
    description: "Templates and apps using Unkey",
    openGraph: {
        title: "Templates | Unkey",
        description: "Templates and apps using Unkey",
        url: "https://unkey.dev/templates",
        siteName: "unkey.dev",
        images: [
            {
                url: "https://unkey.dev/og.png",
                width: 1200,
                height: 675,
            },
        ],
    },
    twitter: {
        title: "Templates | Unkey",
        card: "summary_large_image",
    },
    icons: {
        shortcut: "/unkey.png",
    },
};



type Props = {
    params: {
        slug: string;
    }
}


export default async function Templates(props: Props) {
    const template = templates[props.params.slug]
    if (!template) {
        return notFound()
    }


    const tags: Record<string, string> = {
        Framework: frameworks[template.framework],
        Language: languages[template.language],
    }


    const readme = await fetch(template.readmeUrl).then(res => res.text())
    
    return (
        <>

            <Container>
                <div className="max-w-2xl px-4 py-24 mx-auto sm:px-6 sm:py-32 lg:max-w-7xl lg:px-8">
                    <div className="grid items-center grid-cols-1 gap-x-8 gap-y-16 lg:grid-cols-2">
                        <div>
                            <div className="pb-10 border-b border-gray-200">
                                <h2 className="font-medium text-gray-500">{template.title}</h2>
                                <p className="mt-2 text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">{template.description} </p>
                            </div>

                            <dl className="mt-10 space-y-10">
                                {Object.entries(tags).map(([key, value]) => (

                                    <div key={key}>
                                    <dt className="text-sm font-medium text-gray-900">{value}</dt>
                                    <dd className="mt-3 text-sm text-gray-500">{key}</dd>
                                </div>
                                    ))}

                            </dl>
                        </div>

                        <div>
                            <div className="overflow-hidden bg-gray-100 rounded-lg aspect-h-1 aspect-w-1">
                                <img
                                    src={template.image}
                                    alt="Black kettle with long pour spot and angled body on marble counter next to coffee mug and pour-over system."
                                    className="object-cover object-center w-full h-full"
                                />
                            </div>

                        </div>
                    </div>
                </div>

            </Container>
            <Container>
                <ReactMarkdown remarkPLugins={[remarkGfm]}>
                    {readme}
                </ReactMarkdown>
            </Container>

            
        </>
    );
}
