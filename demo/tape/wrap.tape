# marg wrap tape — shows max_width config in a wide terminal.
# Generates two screenshots side-by-side in story order.
#
# Run from the marg repo root:
#   bash demo/setup.sh
#   vhs demo/tape/wrap.tape

Output "/tmp/marg-vhs-wrap.gif"

Set Shell "bash"
Set FontSize 14
Set Width 2200
Set Height 600
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

# Build two HOME dirs so the same `marg` binary picks up two different configs.
Hide
Type "rm -rf /tmp/marg-home-nocap /tmp/marg-home-cap"
Enter
Type "mkdir -p /tmp/marg-home-nocap /tmp/marg-home-cap/.config/marg"
Enter
Type "printf 'max_width = 80\n' > /tmp/marg-home-cap/.config/marg/config.toml"
Enter
Type "cd /tmp/marg-demo"
Enter
Type "clear"
Enter
Show

# Without cap — content stretches across the whole wide terminal.
Type "HOME=/tmp/marg-home-nocap marg notes/long-form.md"
Enter
Sleep 1s
Screenshot "assets/screenshots/wrap-without-cap.png"
Type ":q"
Enter
Sleep 400ms

# With cap — same file, content held at 80 columns regardless of terminal width.
Type "clear"
Enter
Sleep 200ms
Type "HOME=/tmp/marg-home-cap marg notes/long-form.md"
Enter
Sleep 1s
Screenshot "assets/screenshots/wrap-with-cap.png"
Type ":q"
Enter
Sleep 400ms
