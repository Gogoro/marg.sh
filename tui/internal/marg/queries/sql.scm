; tree-sitter-sql highlight query.
; Keywords aren't matched here — tree-sitter queries can't filter by node
; *type*, only by captured text — so SQL's 350+ `keyword_*` node types are
; coloured by an AST-walk pass in the runner instead.

(comment) @comment
(marginalia) @comment

((literal) @string (#match? @string "^'"))
((literal) @string (#match? @string "^\""))
((literal) @number (#match? @number "^[0-9]"))

(relation (object_reference (identifier) @type))

(field (identifier) @variable.member)

(invocation (object_reference (identifier) @function))
