package justin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/delay"
	"google.golang.org/appengine/urlfetch"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func BuildJustinCommandHandler(expectedCommand string, expectedToken string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := appengine.NewContext(req)
		if (expectedCommand != "" && req.PostFormValue("command") != expectedCommand) || (expectedToken != "" && req.PostFormValue("token") != expectedToken) {
			http.Error(w, `"Forbidden"`, http.StatusForbidden)
			return
		}
		response := fmt.Sprintf("Here's what I found:\n\nhttps://www.google.com/#q=%s", url.QueryEscape(strings.TrimSpace(req.PostFormValue("text"))))
		if strings.HasSuffix(req.PostFormValue("text"), "?") {
			response = fmt.Sprintf("Great question, @%s! %s", req.PostFormValue("user_name"), response)
		} else {
			response = fmt.Sprintf("You got it, @%s! %s", req.PostFormValue("user_name"), response)
		}
		respBytes, _ := buildSlackJSON(ctx, response, true)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"response_type": "in_channel"}`))
		var laterFunc = delay.Func("key", func(delayCtx context.Context, x string) {
			time.Sleep(500 * time.Millisecond)
			sendSlackJSON(delayCtx, req.PostFormValue("response_url"), respBytes)
		})
		laterFunc.Call(ctx, "")
	}
}

func buildSlackJSON(ctx context.Context, text string, isPublic bool) ([]byte, error) {
	type JustinResponse struct {
		ResponseType string `json:"response_type"`
		Text         string `json:"text"`
	}
	justinResponse := &JustinResponse{
		Text: text,
	}
	if isPublic {
		justinResponse.ResponseType = "in_channel"
	} else {
		justinResponse.ResponseType = "ephemeral"
	}
	return json.Marshal(justinResponse)
}

func sendSlackJSON(ctx context.Context, url string, requestBytes []byte) {
	client := urlfetch.Client(ctx)
	slackResponse, _ := client.Post(url, "application/json", bytes.NewReader(requestBytes))
	slackResponse.Body.Close()
}
