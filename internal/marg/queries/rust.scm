; tree-sitter-rust highlight query.

[
  "fn" "let" "const" "static" "if" "else" "match"
  "while" "for" "loop" "return" "break" "continue"
  "use" "pub" "mod" "struct" "enum" "trait" "impl" "type" "where"
  "as" "in" "move" "async" "await" "dyn" "unsafe" "extern"
] @keyword

(mutable_specifier) @keyword

(boolean_literal) @constant.builtin

(line_comment) @comment
(block_comment) @comment

(string_literal) @string
(string_content) @string
(char_literal) @string
(raw_string_literal) @string
(escape_sequence) @string.escape

(integer_literal) @number
(float_literal) @number

(type_identifier) @type
(primitive_type) @type
(field_identifier) @variable.member

(function_item name: (identifier) @function)
(call_expression function: (identifier) @function)
(call_expression function: (field_expression field: (field_identifier) @function))
(macro_invocation macro: (identifier) @function)

(lifetime) @attribute
