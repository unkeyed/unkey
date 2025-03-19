import { ArrowRightLeft, CircleDot, Flame, MessageSquare } from "lucide-react";

export const DetailsComponent = () => {
  return (
    <div className="space-y-8">
      <div className="space-y-4">
        <h1 className="bg-gradient-to-br text-pretty text-transparent bg-gradient-stop bg-clip-text from-white via-white via-30% to-white/30 max-w-sm sm:max-w-none xl:max-w-lg font-medium text-[32px] leading-none sm:text-[56px] md:text-[64px] xl:text-[64px] tracking-tighter">
          One year free for YC W25
        </h1>
        <p className="text-lg text-gray-400">
          Alumni batches get one year at 50% off. Eligibility expires after raising $5 million.
        </p>
      </div>

      <div className="space-y-6">
        {/* Benefit 1 */}
        <div className="space-y-2">
          <div className="flex items-start gap-3">
            <CircleDot className="h-5 w-5 text-orange-500 mt-0.5 shrink-0" />
            <span className="font-medium">Pro plan at any scale</span>
          </div>
          <p className="text-gray-400 pl-8">
            No catch. We want you to succeed in your journey, so we're happy to foot the bill for a
            year.
          </p>
        </div>

        {/* Benefit 2 */}
        <div className="space-y-2">
          <div className="flex items-start gap-3">
            <MessageSquare className="h-5 w-5 text-orange-500 mt-0.5 shrink-0" />
            <span className="font-medium">Priority support</span>
          </div>
          <p className="text-gray-400 pl-8">
            We know Startup life is about moving fast and we won't block you. We give you a
            dedicated slack channel to ask questions and get help.
          </p>
        </div>

        {/* Benefit 3 */}
        <div className="space-y-2">
          <div className="flex items-start gap-3">
            <Flame className="h-5 w-5 text-orange-500 mt-0.5 shrink-0" />
            <span className="font-medium">Concierge onboarding</span>
          </div>
          <p className="text-gray-400 pl-8">
            If helpful for your startup, we can schedule a 1:1 onboarding session to help you get
            started with Unkey and help decide on what's best for your use case.
          </p>
        </div>

        {/* Benefit 4 */}
        <div className="space-y-2">
          <div className="flex items-start gap-3">
            <ArrowRightLeft className="h-5 w-5 text-orange-500 mt-0.5 shrink-0" />
            <span className="font-medium">Migration support</span>
          </div>
          <p className="text-gray-400 pl-8">
            Hands-on support to help you migrate to Unkey from your existing API management
            platform.
          </p>
        </div>
      </div>
    </div>
  );
};
