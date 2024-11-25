package widget

import (
	"context"
	"html/template"
	"time"

	"github.com/glanceapp/glance/internal/assets"
	"github.com/glanceapp/glance/internal/feed"
)

type Lives struct {
	widgetBase      `yaml:",inline"`
	ChannelsRequest []feed.ChannelRequest `yaml:"channels"`
	Channels        []feed.Channel        `yaml:"-"`
	CollapseAfter   int                   `yaml:"collapse-after"`
	SortBy          string                `yaml:"sort-by"`
}

func (widget *Lives) Initialize() error {
	widget.
		withTitle("Twitch Channels").
		withTitleURL("https://www.twitch.tv/directory/following").
		withCacheDuration(time.Minute * 10)

	if widget.CollapseAfter == 0 || widget.CollapseAfter < -1 {
		widget.CollapseAfter = 5
	}

	if widget.SortBy != "viewers" && widget.SortBy != "live" {
		widget.SortBy = "viewers"
	}

	return nil
}

func (widget *Lives) Update(ctx context.Context) {
	channels, err := feed.FetchChannels(widget.ChannelsRequest)

	if !widget.canContinueUpdateAfterHandlingErr(err) {
		return
	}

	if widget.SortBy == "viewers" {
		channels.SortByViewers()
	} else if widget.SortBy == "live" {
		channels.SortByLive()
	}

	widget.Channels = channels
}

func (widget *Lives) Render() template.HTML {
	return widget.render(widget, assets.TwitchChannelsTemplate)
}
