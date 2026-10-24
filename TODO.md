# TODO

Known Markdown support gaps in `read_lsp`.

## Parser and syntax

- [ ] Parse paragraph structure. Processing is currently line-oriented and
  paragraphs are not represented in the document model.
- [ ] Recognize hard line breaks created with two trailing spaces or a trailing
  backslash.
- [ ] Parse task-list checkboxes (`- [ ]` and `- [x]`) as list metadata. Only
  completion snippets for task items are implemented.
- [ ] Parse indented code blocks. Four-space and tab-indented lines are
  currently skipped by inline diagnostics but are not represented as code.
- [ ] Support Setext headings inside list items. They currently work at the top
  level and inside blockquotes, but not in list items.
- [ ] Parse emphasis spans as structured italic, bold, and bold-italic content.
  Current support only reports unmatched `*` and `_` delimiters.
- [ ] Recognize block-level HTML and ignore Markdown inside HTML blocks. Inline
  tags are currently masked, but Markdown between tags is still processed.

## Editor features

- [ ] Add Setext heading snippets to completion. Completion currently offers
  only ATX heading snippets.
- [ ] Fix duplicate documentation for completion options in Neovim. Pressing
  `Shift+K` currently shows the same hover documentation twice.
