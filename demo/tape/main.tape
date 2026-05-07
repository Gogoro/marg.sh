# marg main tape — generates the README screenshots in one run.
# Run from the marg repo root:
#   bash demo/setup.sh
#   vhs demo/tape/main.tape

Output "/tmp/marg-vhs-main.gif"

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

# move into the demo vault and clear the prompt
Hide
Type "cd /tmp/marg-demo"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 200ms
Show

# launch marg on the vault — opens the file tree (folders collapsed)
Type "marg ."
Enter
Sleep 800ms

# expand journal/ so the tree shows nested structure
Enter
Sleep 400ms
Screenshot "assets/screenshots/file-tree.png"

# move the cursor onto the second journal entry and open it
Down
Sleep 100ms
Down
Sleep 200ms
Enter
Sleep 800ms
Screenshot "assets/screenshots/editor-prose.png"

# fuzzy picker for a different file with rich markdown variety
Ctrl+p
Sleep 500ms
Type "launch"
Sleep 600ms
Screenshot "assets/screenshots/fuzzy-picker.png"
Enter
Sleep 800ms
Screenshot "assets/screenshots/markdown-styling.png"

# back to the journal entry to demo visual-line selection
Ctrl+p
Sleep 400ms
Type "04-12"
Sleep 400ms
Enter
Sleep 600ms

# enter visual line mode and extend selection across a few lines
Type "j"
Sleep 100ms
Type "j"
Sleep 100ms
Type "V"
Sleep 200ms
Type "j"
Sleep 100ms
Type "j"
Sleep 100ms
Type "j"
Sleep 400ms
Screenshot "assets/screenshots/visual-line-select.png"
Escape
Sleep 200ms

# clean exit
Type ":q"
Enter
Sleep 400ms
