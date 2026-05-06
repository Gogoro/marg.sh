; tree-sitter-go highlight query.

[
  "package" "import" "func" "var" "const" "type" "struct" "interface"
  "if" "else" "for" "return" "go" "defer" "switch" "case" "default"
  "break" "continue" "fallthrough" "goto" "range" "select" "chan" "map"
] @keyword

[(true) (false) (nil) (iota)] @constant.builtin

(comment) @comment

(interpreted_string_literal) @string
(raw_string_literal) @string
(rune_literal) @string
(escape_sequence) @string.escape

(int_literal) @number
(float_literal) @number
(imaginary_literal) @number

(type_identifier) @type
(package_identifier) @type

(function_declaration name: (identifier) @function)
(method_declaration name: (field_identifier) @function)
(call_expression function: (identifier) @function)
(call_expression function: (selector_expression field: (field_identifier) @function))

(field_identifier) @variable.member
