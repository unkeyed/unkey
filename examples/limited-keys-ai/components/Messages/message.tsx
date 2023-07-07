interface MessageProps {
  role: string;
  content: string;
}
const Message = ({ role, content }: MessageProps) => {
  return (
    <div
      className={`flex  flex-col justify-center mb-2  items-start p-8   rounded-md shadow-md w-1/2 h-auto  max-w-full cursor-pointer [&:not(:first-of-type)]:mt-4 ${
        role !== "user" ? "bg-white" : "ml-auto bg-blue-300"
      }
     
        `}
    >
      <p className="text-lg text-gray-600 mt-0.5  md:text-xl ">{content}</p>
    </div>
  );
};
export default Message;
