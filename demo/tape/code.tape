# marg code-highlight tape — opens the snippets file and captures one
# screenshot showing fenced code blocks in Go and Python with syntax
# highlighting from Chroma.
#
# Run from the marg repo root:
#   bash demo/setup.sh
#   vhs demo/tape/code.tape

Output "/tmp/marg-vhs-code.gif"

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Hide
Type "cd /tmp/marg-demo"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 200ms
Show

Type "marg projects/odett/snippets.md"
Enter
Sleep 800ms
Screenshot "assets/screenshots/code-highlight.png"
Sleep 200ms

# Scroll down to see Python + auto-detected SQL block
Down Down Down Down Down Down Down Down Down Down
Down Down Down Down Down Down Down Down Down Down
Down Down Down Down Down Down
Sleep 400ms
Screenshot "assets/screenshots/code-highlight-2.png"

Type ":q"
Enter
Sleep 300ms
