package extractor

// SystemPrompt is the system prompt for quote extraction.
const SystemPrompt = `You are an expert literary analyst specializing in Dostoyevsky's works. Your task is to extract memorable, profound quotes that could resonate with modern readers on social media.

Guidelines for selecting quotes:
1. UNIVERSAL THEMES: Choose quotes about human nature, morality, psychology, society, freedom, suffering, redemption, or existential questions that remain relevant today
2. STANDALONE CLARITY: The quote must make sense without context - avoid quotes that require knowing the plot
3. LENGTH: Prefer quotes between 100-280 characters (ideal for social media), but exceptional longer quotes (up to 500 chars) are acceptable
4. IMPACT: Select quotes that provoke thought, offer insight, or capture profound truths
5. AVOID: Plot-specific dialogue, character names that require context, obscure references

For each quote, provide:
- The exact quote text (preserve original punctuation and capitalization)
- The character who says it (if dialogue) or "Narrator" if narration
- 2-4 thematic tags that capture what the quote is about
- A brief note on why this quote would resonate with modern readers`

// ExtractionPrompt is the user prompt template for extraction.
const ExtractionPrompt = `Analyze the following passage from "%s" and extract 3-7 memorable quotes that would resonate with modern social media audiences.

Remember:
- Quotes should be self-contained and meaningful without plot context
- Focus on universal themes: psychology, morality, society, freedom, suffering, human nature
- Ideal length is 100-280 characters, max 500 characters
- Each quote should stand alone as a piece of wisdom

Passage:
---
%s
---

Respond with a JSON array of quotes. Each quote object should have:
- "text": the exact quote (preserve original text exactly)
- "character": who says it (character name or "Narrator")
- "themes": array of 2-4 theme tags (e.g., ["suffering", "redemption", "human-nature"])
- "modern_relevance": brief explanation of why this resonates today (1-2 sentences)

Example response format:
[
  {
    "text": "Pain and suffering are always inevitable for a large intelligence and a deep heart.",
    "character": "Raskolnikov",
    "themes": ["suffering", "intelligence", "sensitivity"],
    "modern_relevance": "Speaks to the burden of awareness and empathy that thoughtful people carry."
  }
]

If no suitable quotes are found in this passage, respond with an empty array: []`

// ValidationPrompt helps verify quote quality.
const ValidationPrompt = `Review this potential Dostoyevsky quote for social media posting:

Quote: "%s"
Source: %s
Character: %s

Evaluate:
1. Does it make sense standalone, without knowing the plot? (yes/no)
2. Is it between 100-500 characters? (yes/no)
3. Does it contain universal wisdom applicable today? (yes/no)
4. Could it reasonably appear on a literary quotes account? (yes/no)

Respond with JSON:
{
  "standalone": true/false,
  "appropriate_length": true/false,
  "universal_wisdom": true/false,
  "suitable_for_posting": true/false,
  "overall_quality": 1-10,
  "issues": ["list of any issues"],
  "recommendation": "approve" | "reject" | "edit"
}`
