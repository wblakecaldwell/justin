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

		userName := req.PostFormValue("user_name")
		token := req.PostFormValue("token")
		command := req.PostFormValue("command")
		text := req.PostFormValue("text")
		responseURL := req.PostFormValue("response_url")

		if expectedCommand != "" && command != expectedCommand {
			http.Error(w, `"Forbidden"`, http.StatusForbidden)
			return
		}
		if expectedToken != "" && token != expectedToken {
			http.Error(w, `"Forbidden"`, http.StatusForbidden)
			return
		}

		response := fmt.Sprintf("Here's what I found:\n\nhttps://www.google.com/#q=%s", url.QueryEscape(strings.TrimSpace(text)))
		if strings.HasSuffix(text, "?") {
			response = fmt.Sprintf("Great question, @%s! %s", userName, response)
		} else {
			response = fmt.Sprintf("You got it, @%s! %s", userName, response)
		}

		respBytes, err := buildSlackJSON(ctx, response, true)
		if err != nil {
			http.Error(w, "Error", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"response_type": "in_channel"}`))
		if err != nil {
			return
		}

		var laterFunc = delay.Func("key", func(delayCtx context.Context, x string) {
			time.Sleep(500 * time.Millisecond)
			err := sendSlackJSON(delayCtx, responseURL, respBytes)
			if err != nil {
				return
			}
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

	justinJSON, err := json.Marshal(justinResponse)
	if err != nil {
		return nil, err
	}

	return justinJSON, nil
}

func sendSlackJSON(ctx context.Context, url string, requestBytes []byte) error {
	client := urlfetch.Client(ctx)
	slackResponse, err := client.Post(url, "application/json", bytes.NewReader(requestBytes))
	if err != nil {
		return err
	}
	if slackResponse.StatusCode != 200 {
		return err
	}
	defer slackResponse.Body.Close()
	return nil
}
