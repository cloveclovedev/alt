# Nutrition Estimation Accuracy Improvement

## Problem

LLM-based calorie/protein estimation in `nutrition-check-cloud` tends to overestimate, likely because the model defaults to American portion sizes and recipes (e.g., an American sandwich vs. a Japanese one). The current skill has no guidance on locale-specific estimation or leveraging existing nutrition data sources.

## Solution

Replace the single LLM estimation step in Phase 3c step 4 ("Unknown text or image") with a 3-priority estimation pipeline. Each priority level is tried in order; the first match wins.

## Estimation Priority Pipeline

### Priority 1: Nutrition Label Read

When an uploaded image contains a visible nutrition facts label (栄養成分表示), read the values directly from the label. This is the most accurate source.

- Detect nutrition labels in images before attempting food estimation
- Extract calories (kcal) and protein (g) from the label
- `estimated_by = "label_read"`

### Priority 2: Web Lookup of Official Values

When a brand name, store name, or specific product name is identifiable from the text or image, search the web for official nutrition information.

- Search for `"<product/menu name>" "<brand/store>" 栄養成分` or similar queries
- Use the official value if found on the brand's website, product page, or reliable nutrition database
- `estimated_by = "web_lookup"`

### Priority 3: Web-Search-Assisted LLM Estimation

When neither a nutrition label nor a specific brand/product is identifiable, estimate using web search results as reference data.

1. Search the web for `"<food name>" カロリー 栄養素` to find standard nutritional values for similar dishes
2. Use the search results as reference, combined with the photo's apparent portion size, to estimate calories and protein
3. Always assume Japanese food portions and recipes (not American)
4. When uncertain, estimate conservatively (do not overestimate)
5. `estimated_by = "llm"`

## Changes Required

Only `nutrition-check-cloud/SKILL.md` Phase 3c step 4 needs to be updated. No schema changes, no new dependencies.

## Out of Scope

- User feedback loop for correcting estimates (may revisit later)
- Public nutrition API integration (no suitable free API found; fooddb.mext.go.jp has no public API)
