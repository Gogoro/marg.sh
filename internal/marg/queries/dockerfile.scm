; tree-sitter-dockerfile highlight query.

(comment) @comment

(double_quoted_string) @string
(single_quoted_string) @string
(json_string) @string

(image_spec (image_tag) @string)
(image_spec (image_digest) @string)
(image_spec (image_name) @type)

(env_pair) @property
(label_pair) @property

(variable) @variable
(expansion) @variable
