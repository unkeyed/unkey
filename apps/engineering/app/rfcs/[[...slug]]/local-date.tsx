"use client";

type Props = {
  date: Date | string;
};

export const LocalDate: React.FC<Props> = (props) => {
  const date = typeof props.date === "string" ? new Date(props.date) : props.date;

  return <span suppressHydrationWarning>{date.toLocaleDateString()}</span>;
};
