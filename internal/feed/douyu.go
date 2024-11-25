package feed

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const douyuQueryRoomInfoEndpoint = "https://www.douyu.com/betard/"

func FetchDouyuChannels(rooms []string) (Channels, error) {
	result := make(Channels, 0, len(rooms))

	job := newJob(fetchChannelFromDouyuTask, rooms).withWorkers(10)
	channels, errs, err := workerPoolDo(job)

	if err != nil {
		return result, err
	}

	var failed int

	for i := range channels {
		if errs[i] != nil {
			failed++
			slog.Warn("failed to fetch twitch channel", "channel", rooms[i], "error", errs[i])
			continue
		}

		result = append(result, channels[i])
	}

	if failed == len(rooms) {
		return result, ErrNoContent
	}

	if failed > 0 {
		return result, fmt.Errorf("%w: failed to fetch %d channels", ErrPartialContent, failed)
	}

	return result, nil
}

func fetchChannelFromDouyuTask(channel string) (Channel, error) {
	result := Channel{
		Platform: "www.douyu.com/room/share",
		Login:    strings.ToLower(channel),
	}

	request, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", douyuQueryRoomInfoEndpoint, channel), nil)
	// 设置请求头
	request.Header.Add("accept", "application/json, text/plain, */*")
	request.Header.Add("accept-language", "zh-CN,zh;q=0.9")
	request.Header.Add("baggage", "sentry-environment=master,sentry-public_key=24a6d01353cd4f4691d473a3918377ff,sentry-trace_id=b6ac646a04824e7d8c3d24d731dea012,sentry-sample_rate=0.0001,sentry-sampled=false")
	request.Header.Add("cache-control", "no-cache")
	request.Header.Add("pragma", "no-cache")
	request.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	response, err := decodeJsonFromRequest[DouyuRoomInfoResponse](defaultClient, request)

	if err != nil {
		return result, err
	}

	result.Name = response.Room.RoomName
	result.Exists = true
	result.AvatarUrl = response.Room.Avatar.Small

	if response.Room.ShowStatus != 0 {
		result.IsLive = true
		result.Category = response.Room.SecondLvlName
		timestamp := response.Room.ShowTime
		t := time.Unix(timestamp, 0)

		result.LiveSince = t
	}

	return result, nil
}

type DouyuRoomInfoResponse struct {
	Room struct {
		Avatar        Avatar `json:"avatar"`
		ShowStatus    int    `json:"show_status"`
		RoomID        int    `json:"room_id"`
		Status        string `json:"status"`
		Nickname      string `json:"nickname"`
		ChatLevel     bool   `json:"chat_level"`
		RoomName      string `json:"room_name"`
		Share         Share  `json:"share"`
		SecondLvlName string `json:"second_lvl_name"`
		ShowTime      int64  `json:"show_time"`
	} `json:"room"`
}

type Avatar struct {
	Big    string `json:"big"`
	Middle string `json:"middle"`
	Small  string `json:"small"`
}

type Share struct {
	Video  string `json:"video"`
	Flash  string `json:"flash"`
	Common string `json:"common"`
}
