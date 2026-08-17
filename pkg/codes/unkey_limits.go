package codes

// limitsCustomDomain defines errors related to the workspace's custom domain allowance.
type limitsCustomDomain struct {
	// Exceeded indicates the workspace has already attached as many custom domains as its plan allows.
	Exceeded Code
}

// UnkeyLimitsErrors defines all plan allowance errors in the Unkey system. These
// are distinct from rate limits: the caller cannot retry their way out of one,
// they have to change their plan or free up an existing resource.
type UnkeyLimitsErrors struct {
	// CustomDomain contains errors related to the custom domain allowance.
	CustomDomain limitsCustomDomain
}

// Limits contains all predefined plan allowance error codes. These errors can be
// referenced directly (e.g., codes.Limits.CustomDomain.Exceeded) for consistent
// error handling throughout the application.
var Limits = UnkeyLimitsErrors{
	CustomDomain: limitsCustomDomain{
		Exceeded: Code{SystemUnkey, CategoryUnkeyLimits, "custom_domain_limit_exceeded"},
	},
}
