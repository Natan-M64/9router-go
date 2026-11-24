package chat

import "regexp"

// finishReasonRe matches the OpenAI-format "finish_reason" the gateway sends to
// the client, in a stream chunk or a whole response body.
var finishReasonRe = regexp.MustCompile(`"finish_reason"\s*:\s*"([A-Za-z_]+)"`)

// lastFinishReason returns the last non-empty finish_reason found in the
// response the client received, or "" when there is none. It reads the
// client-facing OpenAI format, so it is the same for every provider; the stored
// response text is cut at a fixed length, which is why this reads the buffer
// before that cut.
func lastFinishReason(response string) string {
	all := finishReasonRe.FindAllStringSubmatch(response, -1)
	if len(all) == 0 {
		return ""
	}
	return all[len(all)-1][1]
}

// requestedMaxTokens returns the output limit the client asked for
// (max_tokens, else max_completion_tokens), or 0 when it set none.
func requestedMaxTokens(maxTokens, maxCompletionTokens *int) int {
	if maxTokens != nil && *maxTokens > 0 {
		return *maxTokens
	}
	if maxCompletionTokens != nil && *maxCompletionTokens > 0 {
		return *maxCompletionTokens
	}
	return 0
}
