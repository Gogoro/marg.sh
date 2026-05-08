; tree-sitter-lua highlight query. The grammar shipped with
; smacker/go-tree-sitter exposes a limited set of anon strings, so we lean
; on node-name captures and skip the typical "[ \"if\" \"then\" ... ]"
; keyword block — those tokens aren't queryable here.

(comment) @comment

(string) @string
(number) @number
(boolean) @constant.builtin
(nil) @constant.builtin

(function_name (identifier) @function)
(function_call (identifier) @function)

(identifier) @variable
