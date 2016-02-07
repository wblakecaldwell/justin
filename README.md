Justin - Slack Google Linker Bot
================================

Justin was designed for those too busy or lazy to [Google](https://google.com)
something on their own. Instead, while in [Slack](https://slack.com/),
just ask Justin to create a Google link for you! This implementation is designed
to be hosted in [Google's AppEngine](https://appengine.google.com). You'll
set up a [slash command](https://hey-now.slack.com/apps/new/A0F82E8CA-slash-commands)
with Slack, so that every time you type the command, followed by some words, 
Justin will create a Google link for you to search for those words.


Example Usage
-------------

In this example, you've configured the slash command to be `/justin`:

You type:

    /justin How do magnets work?

and Justin replies:

    Great question, @blake! Here's what I found:

    https://www.google.com/#safe=off&q=How+do+magnets+work%3F


Host Your Own Justin
--------------------

If you're the type of person that's excited to use Justin, then you're probably
going to think it's too much work to set up. If this is the case, please ask
a grown-up for help. 

This implementation sets up one HTTP endpoint that Slack will call
whenever someone uses your command. First, you'll need to set up a 
[Go](https://golang.org) HTTPS web server on AppEngine. Since AppEngine
doesn't allow you to have a `main()` method, you might want to set up 
the HTTP endpoint in your `init()` method.

	import (
      "github.com/wblakecaldwell/justin"                                                                 
      "net/http"
    )

    func init() {
      http.HandleFunc("/justin", justin.BuildJustinCommandHandler("/justin", "ES5WDo6YIVHfUo1qFjjKSPFK"))
    }

The two parameters to `BuildJustinCommandHandler` are to ensure your service
is only called by allowed Slack instances, where the first is the command name
you configured in Slack, and the second is the token that Slack generates for
you. If you don't care who uses your service, then leave them as empty strings.


Or, Use My Justin!
------------------

If you're as lazy as I think you are, then just use my Justin. He's not doing much.
Create a slash command with the URL set to `https://blake-sandbox.appspot.com/justin-bot/justin`.
My instance doesn't do any validation, so instead of using the command `/justin`, feel free
to just use the name of your laziest friend or co-worker in your Slack community.