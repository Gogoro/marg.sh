; tree-sitter-typescript highlight query.

[
  "function" "class" "interface" "type" "enum" "namespace"
  "if" "else" "for" "while" "do" "return"
  "const" "let" "var" "new" "async" "await" "yield"
  "typeof" "instanceof" "extends" "implements"
  "import" "export" "from" "as"
  "default" "break" "continue" "switch" "case" "try" "catch" "finally"
  "throw" "in" "of" "static" "get" "set" "delete" "void"
  "readonly" "abstract" "declare" "keyof" "infer" "satisfies"
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
(method_definition name: (property_identifier) @function)
(call_expression function: (identifier) @function)
(call_expression function: (member_expression property: (property_identifier) @function))

(new_expression constructor: (identifier) @type)
(class_declaration name: (type_identifier) @type)
(interface_declaration name: (type_identifier) @type)
(type_alias_declaration name: (type_identifier) @type)
(enum_declaration name: (identifier) @type)

(type_identifier) @type
(predefined_type) @type.builtin

(property_identifier) @variable.member
