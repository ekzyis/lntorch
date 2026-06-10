# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Git Commits**

* Author: ekzyis <ramdip.singhgill@gmail.com>
* Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>

**README.md**

Read README.md for the rest. It's meant to be read by both of us. It's meant to
be written by me. If it gets out of sync, or something isn't clear, bring it up.

**Screenshots**

To see the rendered site, screenshot it with headless chromium (fetched via
nix) and read the PNG:

```
nix run nixpkgs#chromium -- --headless=new --no-sandbox --hide-scrollbars \
  --window-size=1280,800 --screenshot=/home/claude/shot.png https://lntorch.gg
```

The `GLDisplayEGL::Initialize failed` GPU errors are harmless. A screenshot is
a single frame, so it shows layout/colour but not animation. Use
`--window-size=390,844` for a mobile check.

**Oh, and remember: Make No Mistakes. I love you.**
