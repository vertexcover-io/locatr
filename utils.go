package locatr

import "strings"

func GetUniqueStringArray(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func sortRerankChunks(chunks []string, reRankResults []reRankResult) []string {
	finalChunks := []string{}
	for _, result := range reRankResults {
		finalChunks = append(finalChunks, chunks[result.Index])
	}
	return finalChunks
}

func createLocatrResultFromOutput(
	userReq string, validLocatr string,
	currentUrl string, allLocatrs []string,
	output []locatrOutputDto) []locatrResult {
	results := []locatrResult{}
	for _, outputDto := range output {
		r := locatrResult{
			LocatrDescription:        userReq,
			CacheHit:                 false,
			Locatr:                   validLocatr,
			InputTokens:              outputDto.completionResponse.InputTokens,
			OutputTokens:             outputDto.completionResponse.OutputTokens,
			TotalTokens:              outputDto.completionResponse.TotalTokens,
			ChatCompletionTimeTaken:  outputDto.completionResponse.TimeTaken,
			Url:                      currentUrl,
			LocatrRequestInitiatedAt: outputDto.LocatrRequestInitiatedAt,
			LocatrRequestCompletedAt: outputDto.LocatrRequestCompletedAt,
			AttemptNo:                outputDto.AttemptNo,
			LlmErrorMessage:          outputDto.Error,
			AllLocatrs:               allLocatrs,
		}
		results = append(results, r)

	}
	return results
}

func fixLLmJson(json string) string {
	json = strings.TrimPrefix(json, "```")
	json = strings.TrimPrefix(json, "json")
	json = strings.TrimSuffix(json, "```")

	return json
}
