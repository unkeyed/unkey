import ReactModal from "react-modal";
import { Props } from "react-modal";
import { AnimatePresence, motion } from "framer-motion";
import { ReactNode } from "react";
import { Button } from "../ui/button";

interface ModalProps extends Props {
  trigger?: () => ReactNode | string;
  setIsOpen: (isOpen: boolean) => void;
}

export const Modal = ({ trigger, children, setIsOpen, isOpen, ...rest }: ModalProps) => {
  const modalVariants = {
    hidden: {
      opacity: 0,
      scale: 0.8,
    },
    visible: {
      opacity: 1,
      y: 0,
      scale: 1,
    },
  };
  const defaultTrigger = (text: string) => <Button>{text}</Button>;
  function close() {
    setIsOpen(false);
  }
  return (
    <>
      {trigger ? (
        // rome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
        <div onClick={() => setIsOpen(true)}>
          {typeof trigger === "function" ? trigger() : defaultTrigger(trigger)}
        </div>
      ) : null}
      <AnimatePresence>
        {isOpen && (
          <ReactModal
            appElement={document.getElementById("root") as HTMLElement}
            onRequestClose={rest.onRequestClose ?? close}
            className="font-jost flex h-full items-center justify-center border-none outline-none"
            style={{
              overlay: {
                backgroundColor: "rgba(0, 0, 0, 0.8)",
                backdropFilter: "blur(5px)",
              },
            }}
            shouldCloseOnOverlayClick
            isOpen={isOpen}
            {...rest}
          >
            <motion.div
              variants={modalVariants}
              initial="hidden"
              animate="visible"
              exit={{ opacity: 0, transition: { duration: 0.1 } }}
              transition={{
                type: "keyframes",
                delay: 0.1,
                ease: "easeInOut",
                duration: 0.3,
              }}
              className="relative flex flex-col mx-4 md:mx-0 justify-center rounded-md border bg-gradient-to-tr from-gray-50 to-gray-100 px-8  py-6 dark:from-black dark:to-slate-900/20 "
            >
              {children}
            </motion.div>
          </ReactModal>
        )}
      </AnimatePresence>
    </>
  );
};
