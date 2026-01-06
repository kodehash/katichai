package analysis

// AIFileScore represents AI-generated code analysis for a file
type AIFileScore struct {
	FilePath          string            `json:"file_path"`
	TotalLOC          int               `json:"total_loc"`
	AIGeneratedLOC    int               `json:"ai_generated_loc"`
	AIPercentage      float64           `json:"ai_percentage"`
	OverallConfidence float64           `json:"overall_confidence"`
	FunctionScores    []AIFunctionScore `json:"function_scores"`
}

// AIFunctionScore represents AI detection for a single function
type AIFunctionScore struct {
	FunctionName string   `json:"function_name"`
	StartLine    int      `json:"start_line"`
	EndLine      int      `json:"end_line"`
	LOC          int      `json:"loc"`
	AIConfidence float64  `json:"ai_confidence"`
	Indicators   []string `json:"indicators"`
}

// CalculateAIScore calculates AI-generated percentage for a file
func CalculateAIScore(fileAnalysis *FileAnalysis, detector *AICodeDetector) *AIFileScore {
	score := &AIFileScore{
		FilePath:       fileAnalysis.FilePath,
		TotalLOC:       fileAnalysis.Metrics.LinesOfCode,
		FunctionScores: make([]AIFunctionScore, 0),
	}

	if score.TotalLOC == 0 {
		return score
	}

	aiLOC := 0
	weightedConfidence := 0.0

	for _, fn := range fileAnalysis.Functions {
		confidence, indicators := detector.CalculateConfidence(fn, fileAnalysis.Language)

		// Threshold: 0.6 confidence (60%) = likely AI-generated
		// This threshold is higher to reduce false positives from good coding practices
		if confidence >= 0.6 {
			aiLOC += fn.LOC

			score.FunctionScores = append(score.FunctionScores, AIFunctionScore{
				FunctionName: fn.Name,
				StartLine:    fn.StartLine,
				EndLine:      fn.EndLine,
				LOC:          fn.LOC,
				AIConfidence: confidence,
				Indicators:   indicators,
			})
		}

		// Weighted average (all functions contribute)
		weightedConfidence += confidence * float64(fn.LOC)
	}

	// Cap aiLOC at TotalLOC to avoid > 100% scenarios
	if aiLOC > score.TotalLOC {
		aiLOC = score.TotalLOC
	}
	
	score.AIGeneratedLOC = aiLOC

	if score.TotalLOC > 0 {
		score.AIPercentage = float64(aiLOC) / float64(score.TotalLOC) * 100
		
		// Cap at 100% (defensive)
		if score.AIPercentage > 100.0 {
			score.AIPercentage = 100.0
		}
		
		score.OverallConfidence = weightedConfidence / float64(score.TotalLOC)
		
		// Cap confidence at 1.0
		if score.OverallConfidence > 1.0 {
			score.OverallConfidence = 1.0
		}
	}

	return score
}

