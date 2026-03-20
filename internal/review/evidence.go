package review

func selectTopEvidence(evidence []EvidenceChunk) EvidenceChunk {
	top := evidence[0]
	hasRuleHeading := ruleIDPattern.FindString(top.HeadingPath) != ""

	for _, chunk := range evidence[1:] {
		chunkHasRuleHeading := ruleIDPattern.FindString(chunk.HeadingPath) != ""

		if chunkHasRuleHeading && !hasRuleHeading {
			top = chunk
			hasRuleHeading = true
			continue
		}

		if chunkHasRuleHeading == hasRuleHeading && chunk.Score > top.Score {
			top = chunk
		}
	}
	return top
}
