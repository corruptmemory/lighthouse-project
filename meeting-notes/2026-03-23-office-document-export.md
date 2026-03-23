# Office Document Export Options for Linux/Go

**Date:** 2026-03-23
**Context:** At some point ERP users will want "dashboard extracts" — typically
Excel, occasionally Word or PowerPoint. What are our options given the
architecture constraints (single Go binary, no CGo, exec bridge for
non-Go protocols)?

## TL;DR

Excel is 90% of the ask. **excelize** (pure Go) covers it without leaving the
binary. Word and PowerPoint are rarer, more template-driven, and fit the exec
bridge pattern.

## Pure Go (lives in the binary)

### Excel (.xlsx) — excelize

- **Library:** [github.com/qax-os/excelize](https://github.com/qax-os/excelize)
- **Status:** Dominant Go library for Excel. Very actively maintained, no CGo.
- **Capabilities:** Read/write `.xlsx`, charts, pivot tables, conditional
  formatting, images, frozen panes, auto-filter, streaming writes for large
  datasets.
- **Fit:** Excellent. Covers the primary export use case entirely in pure Go.

### Word (.docx) — limited options

- **unioffice** ([github.com/unidoc/unioffice](https://github.com/unidoc/unioffice)) —
  Was open source, now commercial (UniDoc per-seat/per-server license). Can do
  Word + Excel + PowerPoint but the licensing cost may not be justified.
- **go-docx** ([github.com/lukasjarosch/go-docx](https://github.com/lukasjarosch/go-docx)) —
  Template-based: create a `.docx` with placeholders, library fills them in.
  Good for structured reports, not for dynamic layout.
- **Roll your own** — `.docx` is a ZIP of XML files. For simple documents
  (title, table, paragraphs) you can template the XML directly. Narrow but
  viable for limited use cases.

### PowerPoint (.pptx) — thin pickings

- No excelize equivalent for PowerPoint in the Go ecosystem.
- unioffice (commercial) or roll-your-own ZIP-of-XML are the only options.

## Exec Bridge (JSON in, file out)

These fit the existing architecture pattern for non-Go protocols.

### Python + openpyxl / python-docx / python-pptx

- Most capable option across all three formats.
- All three libraries are mature and well-documented.
- A thin Python wrapper reads a JSON spec from stdin, writes a file to a
  specified path. Fits the exec bridge pattern exactly.
- Best option for PowerPoint specifically (python-pptx is excellent).

### LibreOffice headless

- `libreoffice --headless --convert-to xlsx template.ods`
- Can convert from ODS/ODT/ODP templates. Heavyweight dependency but extremely
  capable for complex formatting that's easier to design visually than in code.
- Useful if we need pixel-perfect branded report templates.

### Pandoc

- Excellent for Markdown to Word/PDF conversion.
- Not useful for spreadsheets.
- Interesting angle: our markdown-as-artifact widget content could go through
  `widget.md -> Pandoc -> .docx` or `widget.md -> Pandoc -> .pdf` as a
  natural export pipeline.

### wkhtmltopdf / similar

- Render HTML to PDF. Useful if dashboard widgets already render as HTML.
- Could pair with excelize: HTML dashboard view for screen, Excel for data
  export, PDF for print.

## Recommended Approach

| Format     | Approach                         | Rationale                                        |
|------------|----------------------------------|--------------------------------------------------|
| Excel      | excelize (pure Go, in-binary)    | Primary export format, excellent library, no CGo  |
| Word       | Pandoc exec bridge or go-docx    | Markdown -> docx fits existing artifact pattern   |
| PowerPoint | Python exec bridge (python-pptx) | Nothing viable in pure Go                         |
| PDF        | Pandoc or wkhtmltopdf bridge     | HTML/markdown -> PDF for print-ready output       |

## Decision: Start with Excel

Excel is what business users mean when they say "export." excelize is pure Go,
fits all architecture constraints, and produces polished spreadsheets. Solve
this one first; Word/PowerPoint/PDF are lower priority and can use exec bridges
when the need arises.

## Open Questions

- Do we need chart export, or just tabular data? (Charts add complexity in
  excelize but are well-supported.)
- Should widget SQL results have a generic "Export to Excel" button, or do we
  build specific report templates per use case?
- For branded reports (letterhead, logos, formatting), is LibreOffice template
  conversion worth the dependency, or can excelize handle the formatting needs?
