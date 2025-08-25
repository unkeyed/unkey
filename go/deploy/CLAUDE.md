# Rules for AI
- All text, ASCII, and code files MUST end with a newline.
- Use `AIDEV-NOTE:`, `AIDEV-TODO:`, `AIDEV-BUSINESS_RULE:`, or `AIDEV-QUESTION:` (all-caps prefix) as anchor comments aimed at AI and developers.
  * **Important:** Before scanning files, always first **grep for existing anchors** `AIDEV-*` in relevant subdirectories.
  * **Update relevant anchors** when modifying associated code.
  * **Keep `AIDEV-*`s**, and seek permission before removing.
- Make sure to add relevant anchor comments, whenever a file or piece of code is:
  * too complex, or
  * very important, or
  * confusing, or
  * could have a bug
- **Never** take shortcuts. Ask the user if they want to take a shortcut.
- **Always** leave the codebase better tested, better documented, and easier to work with for the next developer.
- All environment variables **MUST** follow the format UNKEY_<SERVICE_NAME>_VARNAME
- **Always** prioritize reliability over performance.
- Use `make install` to test, build, and install the binary w/systemd unit from `$SERVICE/contrib/systemd`
- When a service's `*.go` code changes significantly, increase the patch-level version number.

# Service folder structure

The root implied here is `deploy/`

- Systemd unit files etc: `<service>/contrib/systemd`
- Build artifact directory: `<service>/build`
- Service-level makefile: `<service>/Makefile`
- Global makefile: `Makefile`
- Service binary code: `<service>/cmd/<service | command>`

# Service Pillars

Four services make up the pillars of "Unkey Deploy"

- assetmanagerd
- billaged
- builderd
- metald

# SIFFE/Spire

Spire handles mTLS for all service communication
