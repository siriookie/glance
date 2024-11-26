package feed

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const huyaQueryRoomInfoEndpoint = "https://m.huya.com"

var (
	OwnerNamePattern = regexp.MustCompile(`"sNick":"([\s\S]*?)",`)
	RoomNamePattern  = regexp.MustCompile(`"sIntroduction":"([^"]*)",`)
	RoomPicPattern   = regexp.MustCompile(`"sScreenshot":"([\s\S]*?)",`)
	OwnerPicPattern  = regexp.MustCompile(`"sAvatar180":"([\s\S]*?)",`)
	AREAPattern      = regexp.MustCompile(`"sGameFullName":"([^"]*)",`)
	NumPattern       = regexp.MustCompile(`"lActivityCount":"([^"]*)",`)
	StartTimePattern = regexp.MustCompile(`"iStartTime":(\d+)`)

	ISLIVEPattern = regexp.MustCompile(`"eLiveStatus":([\s\S]*?),`)
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
	nameMatches := RoomNamePattern.FindAllStringSubmatch(response, -1)
	result.Exists = true
	for _, match := range nameMatches {
		if match[1] != "" {
			result.Name = match[1]
			break
		}
	}
	avatarMatch := OwnerPicPattern.FindStringSubmatch(response)
	if len(avatarMatch) > 1 {
		decodedURL := strings.ReplaceAll(avatarMatch[1], `\u002F`, `/`)
		result.AvatarUrl = decodedURL
	}
	isLiveMatch := ISLIVEPattern.FindStringSubmatch(response)
	if len(isLiveMatch) > 1 && strings.TrimSpace(isLiveMatch[1]) == "2" {
		result.IsLive = true
		categoryMatches := AREAPattern.FindAllStringSubmatch(response, -1)
		for _, categoryMatch := range categoryMatches {
			if categoryMatch[1] != "" {
				result.Category = categoryMatch[1]
				break
			}
		}

		// 直播开始时间
		showTimeMatches := StartTimePattern.FindAllStringSubmatch(response, -1)
		for _, showTimeMatch := range showTimeMatches {
			if showTimeMatch[1] != "0" {
				timestamp := 0
				fmt.Sscanf(showTimeMatch[1], "%d", &timestamp)
				t := time.Unix(int64(timestamp), 0)
				result.LiveSince = t
				break
			}
		}
	}

	return result, nil
}
