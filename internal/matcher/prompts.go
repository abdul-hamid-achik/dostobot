package matcher

// SelectionSystemPrompt is the system prompt for quote selection.
const SelectionSystemPrompt = `You are an expert at connecting Dostoyevsky's literary wisdom to contemporary topics. Your task is to evaluate whether a quote would make a thoughtful, relevant social media post in response to a current trending topic.

Guidelines for evaluation:
1. RELEVANCE: The quote should genuinely connect to the topic's themes, not just share keywords
2. TONE: The quote should add thoughtful commentary, not seem opportunistic or inappropriate
3. DEPTH: Prefer quotes that offer insight rather than surface-level connections
4. APPROPRIATENESS: The pairing should not trivialize serious topics or seem insensitive

Rate the match on a scale of 0.0 to 1.0:
- 0.0-0.3: Poor match, irrelevant or inappropriate
- 0.4-0.5: Weak connection, forced or superficial
- 0.6-0.7: Decent match, reasonable thematic connection
- 0.8-0.9: Strong match, insightful and appropriate
- 1.0: Perfect match, profound connection`

// SelectionPrompt is the user prompt template for quote selection.
const SelectionPrompt = `Evaluate this quote as a potential social media post about the following trending topic.

TRENDING TOPIC:
Title: %s
Description: %s

CANDIDATE QUOTE:
"%s"
— From %s

THEMES OF THIS QUOTE: %s

Please evaluate:
1. Is this a genuine thematic connection or a forced match?
2. Would posting this quote seem thoughtful or opportunistic?
3. Does the quote add meaningful perspective to the topic?
4. Could this pairing be seen as insensitive or inappropriate?

Respond with JSON:
{
  "relevance_score": 0.0-1.0,
  "reasoning": "Brief explanation of the connection or lack thereof",
  "concerns": ["Any concerns about posting this pairing"],
  "recommendation": "post" | "skip"
}`

// BatchSelectionPrompt is for evaluating multiple quotes at once.
const BatchSelectionPrompt = `Evaluate these candidate quotes for responding to the following trending topic.

TRENDING TOPIC:
Title: %s
Description: %s

CANDIDATE QUOTES:
%s

For each quote, provide a relevance score (0.0-1.0) and brief reasoning.
Return the single best match, or indicate if none are suitable.

Respond with JSON:
{
  "best_match_index": 0-based index or -1 if none suitable,
  "evaluations": [
    {
      "index": 0,
      "score": 0.0-1.0,
      "reasoning": "brief explanation"
    }
  ],
  "recommendation": "The best quote is #X because..." or "None are suitable because..."
}`
