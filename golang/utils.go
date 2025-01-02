package locatr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mafredri/cdp/devtool"
)

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

func filterTargets(pages []*devtool.Target) []*devtool.Target {
	newTargets := []*devtool.Target{}
	for _, target := range pages {
		if target.Type == "page" && !strings.HasPrefix("DevTools", target.Title) {
			newTargets = append(newTargets, target)
		}
	}
	return newTargets
}

func getWebsocketDebugUrl(url string, tabIndex int, ctx context.Context) (string, error) {
	devt := devtool.New(url)
	targets, err := devt.List(ctx)
	if err != nil {
		return "", err
	}
	targetsFiltered := filterTargets(targets)
	for indx, target := range targetsFiltered {
		if indx == tabIndex {
			return target.WebSocketDebuggerURL, nil
		}
	}
	return "", fmt.Errorf("Tab with index %d not present in the browser", tabIndex)
}
