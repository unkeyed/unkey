
// id -> label
export const frameworks = {
    django: "Django",
    nextjs: "Next.js",
    svelte: "Svelte"
}

// id -> label
export const languages = {
    ts: "Typescript",
    py: "Python",
    go: "Golang",
    rs: "Rust"
}


export type Template = {
    title: string
    description: string
    /**
     * GitHub username or similar
     */
    author: string

    /**
     * Url to the repository
     */
    repository: string

    /**
     * Url to the image
     */
    image: string

    /**
     * Url to the raw readme
     */
    readmeUrl: string

    language: keyof typeof languages
    framework: keyof typeof frameworks

}




export const templates: Record<string, Template> = {
    "sveltekit-boilerplate": {
        title: "SvelteKit Boilerplate",
        description: "A SvelteKit app including nested routes, layouts, and page endpoints.",
        author: "chronark",
        repository: "",
        image: "https://vercel.com/_next/image?url=https%3A%2F%2Fimages.ctfassets.net%2Fe5382hct74si%2F5WIYQtnSEfZKYFB9kvsR0w%2F974bee31f87aa376a54dccdb0713629d%2FCleanShot_2022-05-23_at_22.13.20_2x.png&w=1920&q=75&dpl=dpl_7WCNpyWTxwxjZmqLAa9rQb3zYTgz",
        readmeUrl: "https://raw.githubusercontent.com/vercel/vercel/main/examples/nextjs/README.md",
        language: "ts",
        framework: "svelte"

    },
    envshare: {
        title: "Envshare",
        description: "EnvShare is a simple tool to share environment variables securely. It uses AES-GCM to encrypt your data before sending it to the server. The encryption key never leaves your browser.",
        author: "chronark",
        repository: "",
        image: "https://vercel.com/_next/image?url=https%3A%2F%2Fimages.ctfassets.net%2Fe5382hct74si%2F5SaFBHXp5FBFJbsTzVqIJ3%2Ff0f8382369b7642fd8103debb9025c11%2Fenvshare.png&w=1920&q=75&dpl=dpl_7WCNpyWTxwxjZmqLAa9rQb3zYTgz",
        readmeUrl: "",
        language: "ts",
        framework: "nextjs"

    },
    payments: {
        title: "Next.js Subscription Payments Starter",
        description: "The all-in-one starter kit for high-performance SaaS applications.",
        author: "chronark",
        repository: "",
        image: "https://vercel.com/_next/image?url=https%3A%2F%2Fimages.ctfassets.net%2Fe5382hct74si%2F7segt6pEQeFpvw4KiMND8%2F7d9ce0dbc600aeb8890d39ec892cb8f0%2FNew_Project__3_.png&w=1920&q=75&dpl=dpl_7WCNpyWTxwxjZmqLAa9rQb3zYTgz",
        readmeUrl: "",
        language: "ts",
        framework: "nextjs"

    },
}