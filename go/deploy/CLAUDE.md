- **Never** delete anything from this file.
- All non-binary files MUST end with a newline. This is non-negotiable.
- Use `AIDEV-NOTE:`, `AIDEV-TODO:`, `AIDEV-BUSINESS_RULE:`, or `AIDEV-QUESTION:` (all-caps prefix) as anchor comments aimed at AI and developers.
  * **Important:** Before scanning files, always first try to **grep for existing anchors** `AIDEV-*` in relevant subdirectories.
  * **Update relevant anchors** when modifying associated code.
  * **Do not remove `AIDEV-*`s** without explicit human instruction.
- Make sure to add relevant anchor comments, whenever a file or piece of code is:
  * too complex, or
  * very important, or
  * confusing, or
  * could have a bug
- **Never** take shortcuts. Ask the user if they want to take a shortcut.
- **Always** leave the codebase better tested, better documented, and easier to work with for the next developer.
- All environment variables **MUST** follow the format UNKEY_<SERVICE_NAME>_VARNAME
- **Always** prioritize reliability over performance.
- **Never** use `go build` for any of the `deploy/{assetmanagerd,billaged,builderd,metald}` binaries.
- Use `make install` to build and install the binary w/systemd unit from $SERVICE/contrib/systemd
- Use `make build` to test that the binary builds.
