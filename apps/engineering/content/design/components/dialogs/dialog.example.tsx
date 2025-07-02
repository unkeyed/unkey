"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import {
  Button,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  Input,
} from "@unkey/ui";
import { useState } from "react";

export const DialogExample: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [isWarningOpen, setIsWarningOpen] = useState(false);
  const [inputValue, setInputValue] = useState("");

  const handleCloseAttempt = () => {
    // In a real application, you would show a proper confirmation dialog
    const confirmClose = window.confirm(
      "You have unsaved changes. Are you sure you want to close?",
    );
    if (confirmClose) {
      setIsWarningOpen(false);
    }
  };

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-4 w-full max-w-md mx-auto">
  {/* Basic Dialog */}
  <div className="flex flex-col gap-2">
    <h3 className="text-sm font-medium">Basic Dialog</h3>
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline">Open Basic Dialog</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Basic Dialog</DialogTitle>
          <DialogDescription>
            This is a basic dialog with a title and description.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <p>This is the main content of the dialog.</p>
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button>Save Changes</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>

  {/* Controlled Dialog */}
  <div className="flex flex-col gap-2">
    <h3 className="text-sm font-medium">Controlled Dialog</h3>
    <Button variant="outline" onClick={() => setIsOpen(true)}>
      Open Controlled Dialog
    </Button>
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Controlled Dialog</DialogTitle>
          <DialogDescription>
            This dialog is controlled by external state.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <Input
            placeholder="Enter some text..."
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setIsOpen(false)}>
            Cancel
          </Button>
          <Button onClick={() => setIsOpen(false)}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>

  {/* Dialog with Close Warning */}
  <div className="flex flex-col gap-2">
    <h3 className="text-sm font-medium">Dialog with Close Warning</h3>
    <Button variant="outline" onClick={() => setIsWarningOpen(true)}>
      Open Dialog with Warning
    </Button>
    <Dialog open={isWarningOpen} onOpenChange={setIsWarningOpen}>
      <DialogContent showCloseWarning onAttemptClose={handleCloseAttempt}>
        <DialogHeader>
          <DialogTitle>Dialog with Close Warning</DialogTitle>
          <DialogDescription>
            This dialog will show a warning when you try to close it.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <p>Try clicking the X button or pressing Escape to see the warning.</p>
          <Input placeholder="Make some changes..." />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setIsWarningOpen(false)}>
            Cancel
          </Button>
          <Button onClick={() => setIsWarningOpen(false)}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>

  {/* Custom Styled Dialog */}
  <div className="flex flex-col gap-2">
    <h3 className="text-sm font-medium">Custom Styled Dialog</h3>
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline">Open Custom Dialog</Button>
      </DialogTrigger>
      <DialogContent className="max-w-md bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-blue-950 dark:to-indigo-900">
        <DialogHeader>
          <DialogTitle className="text-blue-900 dark:text-blue-100">
            Custom Styled Dialog
          </DialogTitle>
          <DialogDescription className="text-blue-700 dark:text-blue-300">
            This dialog has custom styling applied.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <p className="text-blue-800 dark:text-blue-200">
            The content area can be styled with custom classes.
          </p>
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Close</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>

  {/* Dialog without Footer */}
  <div className="flex flex-col gap-2">
    <h3 className="text-sm font-medium">Dialog without Footer</h3>
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline">Open Simple Dialog</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Simple Dialog</DialogTitle>
          <DialogDescription>This dialog doesn't have a footer section.</DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <p>Sometimes you don't need action buttons in the footer.</p>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</div>`}
    >
      <Row>
        <div className="flex flex-col gap-4 w-full max-w-md mx-auto">
          {/* Basic Dialog */}
          <div className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">Basic Dialog</h3>
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="outline">Open Basic Dialog</Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Basic Dialog</DialogTitle>
                  <DialogDescription>
                    This is a basic dialog with a title and description.
                  </DialogDescription>
                </DialogHeader>
                <div className="py-4">
                  <p>This is the main content of the dialog.</p>
                </div>
                <DialogFooter>
                  <DialogClose asChild>
                    <Button variant="outline">Cancel</Button>
                  </DialogClose>
                  <Button>Save Changes</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          {/* Controlled Dialog */}
          <div className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">Controlled Dialog</h3>
            <Button variant="outline" onClick={() => setIsOpen(true)}>
              Open Controlled Dialog
            </Button>
            <Dialog open={isOpen} onOpenChange={setIsOpen}>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Controlled Dialog</DialogTitle>
                  <DialogDescription>
                    This dialog is controlled by external state.
                  </DialogDescription>
                </DialogHeader>
                <div className="py-4">
                  <Input
                    placeholder="Enter some text..."
                    value={inputValue}
                    onChange={(e) => setInputValue(e.target.value)}
                  />
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setIsOpen(false)}>
                    Cancel
                  </Button>
                  <Button onClick={() => setIsOpen(false)}>Save</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          {/* Dialog with Close Warning */}
          <div className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">Dialog with Close Warning</h3>
            <Button variant="outline" onClick={() => setIsWarningOpen(true)}>
              Open Dialog with Warning
            </Button>
            <Dialog open={isWarningOpen} onOpenChange={setIsWarningOpen}>
              <DialogContent showCloseWarning onAttemptClose={handleCloseAttempt}>
                <DialogHeader>
                  <DialogTitle>Dialog with Close Warning</DialogTitle>
                  <DialogDescription>
                    This dialog will show a warning when you try to close it.
                  </DialogDescription>
                </DialogHeader>
                <div className="py-4">
                  <p>Try clicking the X button or pressing Escape to see the warning.</p>
                  <Input placeholder="Make some changes..." />
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setIsWarningOpen(false)}>
                    Cancel
                  </Button>
                  <Button onClick={() => setIsWarningOpen(false)}>Save</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          {/* Custom Styled Dialog */}
          <div className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">Custom Styled Dialog</h3>
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="outline">Open Custom Dialog</Button>
              </DialogTrigger>
              <DialogContent className="max-w-md bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-blue-950 dark:to-indigo-900">
                <DialogHeader>
                  <DialogTitle className="text-blue-900 dark:text-blue-100">
                    Custom Styled Dialog
                  </DialogTitle>
                  <DialogDescription className="text-blue-700 dark:text-blue-300">
                    This dialog has custom styling applied.
                  </DialogDescription>
                </DialogHeader>
                <div className="py-4">
                  <p className="text-blue-800 dark:text-blue-200">
                    The content area can be styled with custom classes.
                  </p>
                </div>
                <DialogFooter>
                  <DialogClose asChild>
                    <Button variant="outline">Close</Button>
                  </DialogClose>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          {/* Dialog without Footer */}
          <div className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">Dialog without Footer</h3>
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="outline">Open Simple Dialog</Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Simple Dialog</DialogTitle>
                  <DialogDescription>This dialog doesn't have a footer section.</DialogDescription>
                </DialogHeader>
                <div className="py-4">
                  <p>Sometimes you don't need action buttons in the footer.</p>
                </div>
              </DialogContent>
            </Dialog>
          </div>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};
