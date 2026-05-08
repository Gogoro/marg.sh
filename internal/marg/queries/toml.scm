; tree-sitter-toml highlight query.

(comment) @comment

(string) @string
(integer) @number
(float) @number
(boolean) @constant.builtin

(local_date) @constant
(local_date_time) @constant
(local_time) @constant
(offset_date_time) @constant

; Section headers like [server] or [[hosts]].
(table (_) @type)
(table_array_element (_) @type)

; Bare keys read as properties; quoted keys still get the property colour.
(bare_key) @property
(quoted_key) @property
(dotted_key) @property
