"use client"

import { addToWaitlist } from "@/app/addToWaitlist";
import {  toast } from "sonner";
export const Waitlist: React.FC=()=>{
    return (
        <form
                  className="relative"
                  action={async (data: FormData) => {
                    const email = data.get("email");
                    if (!email) {
                      toast.error("You need to enter an email");
                      return;
                    }
                    toast.promise(addToWaitlist(email as string), {
                      loading: "Adding to db",
                      success: (data) =>
                        `Thank you, you're number ${Intl.NumberFormat().format(data)} on the list`,
                      error: "Error",
                    });
                  }}
                >
                  <div className="flex">
                    <div className="flex-1">
                      <input type="email" name="email" id="email" placeholder="Enter email address" className="w-full px-4 py-4 text-base text-gray-900 placeholder-gray-600 bg-white border border-gray-300 border-gray-600 rounded-l-lg focus:ring-gray-900 focus:border-gray-900 caret-gray-900" required />
                    </div>

                    <button type="submit" className="px-10 py-4 text-base font-bold text-white transition-all duration-200 bg-gray-900 border border-transparent rounded-r-lg sm:px-16 focus:outline-none">Enter Waitlist</button>
                  </div>
                </form>
    )
}