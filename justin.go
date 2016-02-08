// Package justin creates an HTTP endpoint to implement the backend of a Slack "slash command"
// which creates Google search links from text, when the Slack user is otherwise unwilling or
// unable to do so. The endpoint is written for hosting on Google AppEngine.
//
// There's no main method in AppEngine sites, so you need to wire up the endpoint in the init() function.
//
// Example usage:
//   import (
//     "github.com/wblakecaldwell/justin"
//     "net/http"
//   )
//
//   func init() {
//     http.HandleFunc("/justin", justin.BuildJustinCommandHandler("/justin", "ES5WDo6YIVHfUo1qFjjKSPFK"))
//   }
package justin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/delay"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BuildJustinCommandHandler builds an HTTP endpoint that responds to a Justin Slack "slash command"
// - command is the expected Slack command, or empty string if we don't care
// - token is the expected Slack token, or empty string if we don't care
func BuildJustinCommandHandler(expectedCommand string, expectedToken string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := appengine.NewContext(req)

		userName := req.PostFormValue("user_name")
		token := req.PostFormValue("token")
		command := req.PostFormValue("command")
		text := req.PostFormValue("text")
		responseURL := req.PostFormValue("response_url")

		log.Debugf(ctx, `Request: userName: %s
token: %s
command: %s
text: %s
team id: %s
team domain: %s
channel id: %s
channel name: %s
user id: %s
response url: %s`, userName, token, command, text, req.PostFormValue("team_id"), req.PostFormValue("team_domain"), req.PostFormValue("channel_id"),
			req.PostFormValue("channel_name"), req.PostFormValue("user_id"), responseURL)

		// Validate if we're supposed to
		if expectedCommand != "" && command != expectedCommand {
			log.Errorf(ctx, "Forbidden - invalid command '%s'; expected '%s'", command, expectedCommand)
			http.Error(w, `"Forbidden"`, http.StatusForbidden)
			return
		}
		if expectedToken != "" && token != expectedToken {
			log.Errorf(ctx, "Forbidden - invalid token '%s'; expected '%s'", token, expectedToken)
			http.Error(w, `"Forbidden"`, http.StatusForbidden)
			return
		}

		// build the Slack response text
		response := fmt.Sprintf("Here's what I found:\n\nhttps://www.google.com/#q=%s", url.QueryEscape(strings.TrimSpace(text)))
		if strings.HasSuffix(text, "?") {
			response = fmt.Sprintf("Great question, @%s! %s", userName, response)
		} else {
			response = fmt.Sprintf("You got it, @%s! %s", userName, response)
		}

		// Create the JSON payload with the response, to send back to Slack
		respBytes, err := buildSlackJSON(ctx, response, true)
		if err != nil {
			// already logged
			http.Error(w, "Error", 500)
			return
		}

		// tell Slack to show the user's command
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"response_type": "in_channel"}`))
		if err != nil {
			log.Errorf(ctx, "Error echoing the request: %s", err)
		}

		// queue up the delayed response
		var laterFunc = delay.Func("key", func(delayCtx context.Context, x string) {
			log.Debugf(delayCtx, "Sending delayed response to Slack for request: %s", text)
			time.Sleep(500 * time.Millisecond)
			err := sendSlackJSON(delayCtx, responseURL, respBytes)
			if err != nil {
				// already logged
				return
			}
			log.Debugf(delayCtx, "Success! Response sent back to Slack: %s", respBytes)
		})
		laterFunc.Call(ctx, "")
	}
}

// buildSlackJSON returns a public or private Slack JSON response as a byte slice.
// Context is passed in for logging.
func buildSlackJSON(ctx context.Context, text string, isPublic bool) ([]byte, error) {
	// JustinResponse is a response we send back to Slack
	type JustinResponse struct {
		ResponseType string `json:"response_type"` // "ephemeral" (private) or "in_channel" (public)
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

	log.Debugf(ctx, "Marshalling JustinResponse: %#v", justinResponse)
	justinJSON, err := json.Marshal(justinResponse)
	if err != nil {
		log.Errorf(ctx, "Error marshalling JSON for JustinResponse: %#v - %s", justinResponse, err)
		return nil, err
	}

	log.Debugf(ctx, "Slack response: %s", string(justinJSON))
	return justinJSON, nil
}

// sendSlackJSON sends the JSON response back to Slack
func sendSlackJSON(ctx context.Context, url string, requestBytes []byte) error {
	client := urlfetch.Client(ctx)
	slackResponse, err := client.Post(url, "application/json", bytes.NewReader(requestBytes))
	if err != nil {
		log.Errorf(ctx, "Error submitting response to Slack: %s", err)
		return err
	}
	if slackResponse.StatusCode != 200 {
		log.Errorf(ctx, "Received status code %d when submitting response to Slack", slackResponse.StatusCode)
		return err
	}
	defer slackResponse.Body.Close()
	return nil
}
