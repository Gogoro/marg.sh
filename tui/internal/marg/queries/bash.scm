; tree-sitter-bash highlight query.

[
  "if" "then" "elif" "else" "fi" "for" "while" "do" "done"
  "case" "esac" "function" "in" "select" "until"
] @keyword

(comment) @comment

(string) @string
(string_content) @string
(raw_string) @string
(ansi_c_string) @string
(heredoc_body) @string

(number) @number

(command_name) @function

(variable_name) @variable
(simple_expansion) @variable
(expansion) @variable

(test_operator) @operator
(file_redirect) @operator
