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
    image: { src: "/images/team/james.jpg" },
  },
  andreas: {
    name: "Andreas Thomas",
    role: "Co-Founder",
    image: { src: "/images/team/andreas.jpeg" },
  },
  wilfred: {
    name: "Wilfred Almeida",
    role: "Freelance Writer",
    image: { src: "/images/blog-images/ocr-post/wilfred.jpg" },
  },
} satisfies { [name: string]: Author };
