; tree-sitter-javascript highlight query.

[
  "function" "class" "if" "else" "for" "while" "do" "return"
  "const" "let" "var" "new" "async" "await" "yield"
  "typeof" "instanceof" "extends" "import" "export" "from" "as"
  "default" "break" "continue" "switch" "case" "try" "catch" "finally"
  "throw" "in" "of" "static" "get" "set" "delete" "void" "with"
] @keyword

[(true) (false) (null) (undefined)] @constant.builtin
(this) @variable.builtin
(super) @variable.builtin

(comment) @comment

(string) @string
(template_string) @string
(string_fragment) @string
(escape_sequence) @string.escape
(regex) @string

(number) @number

(function_declaration name: (identifier) @function)
(function_expression name: (identifier) @function)
(generator_function_declaration name: (identifier) @function)
(method_definition name: (property_identifier) @function)
(call_expression function: (identifier) @function)
(call_expression function: (member_expression property: (property_identifier) @function))
(new_expression constructor: (identifier) @type)

(class_declaration name: (identifier) @type)

(property_identifier) @variable.member
