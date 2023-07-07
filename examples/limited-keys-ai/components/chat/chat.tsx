"use client";

import { useChat } from "ai/react";
import MessageContainer from "../Messages/container";
import Message from "../Messages/message";

const Chat = () => {
  const { messages, input, handleInputChange, handleSubmit } = useChat({
    api: "/api/completions",
  });

  return (
    <MessageContainer>
      {messages && messages.length > 0 ? (
        messages.map((message) => (
          <Message
            key={message.id}
            role={message.role}
            content={message.content}
          />
        ))
      ) : (
        <p>No messages yet...</p>
      )}

      <form
        className=" w-full flex gap-3 justify-center items-center md:left-[1%]"
        onSubmit={handleSubmit}
      >
        {
          <>
            <textarea
              className="w-5/6 p-3 border-2 border-black break-words whitespace-pre-line border-solid text-base"
              placeholder="Message"
              name="message"
              value={input}
              onChange={handleInputChange}
            />
            <button
              type="submit"
              className="bg-green-200 text-black/60 justify-center items-center flex gap-2 mt-2  rounded-md py-2 px-3 mb-2 text-base  md:text-2xl"
            >
              <span> Send</span> <span className="">🚁</span>
            </button>
          </>
        }
      </form>
    </MessageContainer>
  );
};

export default Chat;
