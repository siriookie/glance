package feed

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const huyaQueryRoomInfoEndpoint = "https://www.douyu.com/betard/"

var (
	OwnerNamePattern = regexp.MustCompile(`"sNick":"([\s\S]*?)",`)
	RoomNamePattern  = regexp.MustCompile(`"sIntroduction":"([\s\S]*?)",`)
	RoomPicPattern   = regexp.MustCompile(`"sScreenshot":"([\s\S]*?)",`)
	OwnerPicPattern  = regexp.MustCompile(`"sAvatar180":"([\s\S]*?)",`)
	AREAPattern      = regexp.MustCompile(`"sGameFullName":"([\s\S]*?)",`)
	NumPattern       = regexp.MustCompile(`"lActivityCount":([\s\S]*?),`)
	ISLIVEPattern    = regexp.MustCompile(`"eLiveStatus":([\s\S]*?),`)
)

func FetchHuyaChannels(rooms []string) (Channels, error) {
	result := make(Channels, 0, len(rooms))

	job := newJob(fetchChannelFromHuyaTask, rooms).withWorkers(10)
	channels, errs, err := workerPoolDo(job)

	if err != nil {
		return result, err
	}

	var failed int

	for i := range channels {
		if errs[i] != nil {
			failed++
			slog.Warn("failed to fetch huya channel", "channel", rooms[i], "error", errs[i])
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

func fetchChannelFromHuyaTask(channel string) (Channel, error) {
	result := Channel{
		Platform: "m.huya.com",
		Login:    strings.ToLower(channel),
	}

	request, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", huyaQueryRoomInfoEndpoint, channel), nil)
	// 设置请求头
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 5.0; SM-G900P Build/LRX21T) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Mobile Safari/537.36")

	response, err := doRequest(defaultClient, request)

	if err != nil {
		return result, err
	}

	// 匹配正则并提取信息
	nameMatch := RoomNamePattern.FindStringSubmatch(response)
	if len(nameMatch) > 1 {
		result.Name = nameMatch[1]
		result.Exists = true
	}

	avatarMatch := OwnerPicPattern.FindStringSubmatch(response)
	if len(avatarMatch) > 1 {
		result.AvatarUrl = avatarMatch[1]
	}
	isLiveMatch := ISLIVEPattern.FindStringSubmatch(response)
	if len(isLiveMatch) > 1 && strings.TrimSpace(isLiveMatch[1]) == "1" {
		result.IsLive = true

		// 分类
		categoryMatch := AREAPattern.FindStringSubmatch(response)
		if len(categoryMatch) > 1 {
			result.Category = categoryMatch[1]
		}

		// 直播开始时间
		showTimeMatch := NumPattern.FindStringSubmatch(response)
		if len(showTimeMatch) > 1 {
			timestamp := 0
			fmt.Sscanf(showTimeMatch[1], "%d", &timestamp)
			t := time.Unix(int64(timestamp), 0)
			result.LiveSince = t
		}
	}

	return result, nil
}
