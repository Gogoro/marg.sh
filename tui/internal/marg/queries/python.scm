; tree-sitter-python highlight query.

[
  "def" "class" "if" "elif" "else" "for" "while" "return"
  "import" "from" "as" "try" "except" "finally" "raise"
  "with" "yield" "pass" "break" "continue" "lambda"
  "global" "nonlocal" "and" "or" "not" "in" "is"
  "async" "await" "match" "case"
] @keyword

[(true) (false) (none)] @constant.builtin

(comment) @comment

(string) @string
(escape_sequence) @string.escape

(integer) @number
(float) @number

(function_definition name: (identifier) @function)
(class_definition name: (identifier) @type)

(call function: (identifier) @function)
(call function: (attribute attribute: (identifier) @function))

(decorator (identifier) @attribute)

(type (identifier) @type)
