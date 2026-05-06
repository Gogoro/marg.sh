; tree-sitter-yaml highlight query.

(comment) @comment

(string_scalar) @string
(double_quote_scalar) @string
(single_quote_scalar) @string
(block_scalar) @string

(integer_scalar) @number
(float_scalar) @number
(boolean_scalar) @constant.builtin
(null_scalar) @constant.builtin

; Anchors and aliases (named references)
(anchor_name) @attribute
(alias_name) @attribute

; Tag (e.g. !!str) marks a scalar's type explicitly.
(tag) @type

; Mapping keys: the first child of a block_mapping_pair is the key
; (the second is `value:`). Capture key-position scalars as @property
; so config keys read distinctly from values.
(block_mapping_pair
  key: (flow_node) @property)

(flow_mapping
  (flow_pair
    key: (flow_node) @property))
