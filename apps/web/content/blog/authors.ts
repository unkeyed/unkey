export type Author = {
  name: string;
  role: string;
  image: {
    src: string;
    alt?: string;
  };
};

export const authors = {
  james: {
    name: "James Perkins",
    role: "Co-Founder / CEO",
    image: { src: "/james.jpg" },
  },
  andreas: {
    name: "Andreas Thomas",
    role: "Co-Founder",
    image: { src: "/andreas.jpeg" },
  },
  wilfred: {
    name: "Wilfred Almeida",
    role: "Freelance Writer",
    image: { src: "/images/blog-images/ocr-post/wilfred.jpg" },
  },
} satisfies { [name: string]: Author };
