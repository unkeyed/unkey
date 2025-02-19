export type Author = {
  name: string;
  role: string;
  image: {
    src: string;
    alt?: string;
  };
};

type Authors = {
  [key: string]: Author;
};

export const authors: Authors = {
  dom: {
    name: "Dom Eccleston",
    role: "Engineer",
    image: { src: "/images/team/dom.jpeg" },
  },
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
  michael: {
    name: "Michael Silva",
    role: "Developer",
    image: { src: "/images/team/michael.jpg" },
  },
  oz: {
    name: "Oguzhan Olguncu",
    role: "Developer",
    image: { src: "/images/team/oz.jpeg" },
  },
};
