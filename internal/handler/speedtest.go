package handler

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
)

type SpeedtestHandler struct {
	client *telegram.Client
}

func NewSpeedtestHandler(cli *telegram.Client) *SpeedtestHandler {
	return &SpeedtestHandler{client: cli}
}

func (h *SpeedtestHandler) Handle(ctx context.Context, e tg.Entities, msg *tg.Message) error {
	peer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return err
	}

	sender := message.NewSender(h.client.API())
	b := sender.To(peer).Reply(msg.ID)

	u, err := b.Text(ctx, "Starting speedtest...")
	if err != nil {
		return err
	}

	msgID := extractSpeedtestMsgID(u)

	editMsg := func(text string) {
		sender.To(peer).Edit(msgID).StyledText(ctx, html.String(nil, text))
	}

	client := &http.Client{Timeout: 60 * time.Second}

	editMsg("Running speedtest...")

	// Ping test (quick, 3 samples)
	pings := make([]float64, 0, 3)
	for i := 0; i < 3; i++ {
		start := time.Now()
		resp, err := client.Head("https://speed.cloudflare.com/__down?bytes=0")
		if err == nil {
			pings = append(pings, float64(time.Since(start).Milliseconds()))
			resp.Body.Close()
		}
	}
	avgPing, jitter := calcPingStats(pings)

	// Download test (250MB)
	dlSpeed, dlTime, dlSize, err := runDownloadTest(client, 262144000)
	if err != nil {
		editMsg(fmt.Sprintf("<b>Speedtest Failed</b>\n\nError: %v", err))
		return nil
	}

	// Upload test (25MB)
	ulSpeed, ulTime, ulSize, _ := runUploadTest(client, 26214400)

	serverInfo := fetchServerInfo(client)

	result := buildSpeedtestResult(serverInfo, avgPing, jitter, dlSpeed, dlTime, dlSize, ulSpeed, ulTime, ulSize)
	editMsg(result)

	return nil
}

func extractSpeedtestMsgID(u tg.UpdatesClass) int {
	if up, ok := u.(*tg.UpdateShortSentMessage); ok {
		return up.ID
	}
	if ups, ok := u.(*tg.Updates); ok {
		for _, update := range ups.Updates {
			if m, ok := update.(*tg.UpdateNewMessage); ok {
				if msg, ok := m.Message.(*tg.Message); ok {
					return msg.ID
				}
			}
			if m, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if msg, ok := m.Message.(*tg.Message); ok {
					return msg.ID
				}
			}
		}
	}
	return 0
}

func calcPingStats(pings []float64) (avg, jitter float64) {
	if len(pings) == 0 {
		return 0, 0
	}
	var sum float64
	for _, p := range pings {
		sum += p
	}
	avg = sum / float64(len(pings))

	var sumDiffSq float64
	for _, p := range pings {
		diff := p - avg
		sumDiffSq += diff * diff
	}
	jitter = math.Sqrt(sumDiffSq / float64(len(pings)))
	return
}

func runDownloadTest(client *http.Client, bytes int) (speed, duration float64, size int64, err error) {
	url := fmt.Sprintf("https://speed.cloudflare.com/__down?bytes=%d", bytes)
	start := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	written, _ := io.Copy(io.Discard, resp.Body)
	dur := time.Since(start)

	speed = (float64(written) * 8) / (dur.Seconds() * 1000 * 1000)
	duration = dur.Seconds()
	size = written
	return
}

func runUploadTest(client *http.Client, bytes int) (speed, duration float64, size int64, err error) {
	url := "https://speed.cloudflare.com/__up"
	data := strings.NewReader(strings.Repeat("0", bytes))

	start := time.Now()
	resp, err := client.Post(url, "application/octet-stream", data)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	dur := time.Since(start)
	speed = (float64(bytes) * 8) / (dur.Seconds() * 1000 * 1000)
	duration = dur.Seconds()
	size = int64(bytes)
	return
}

func fetchServerInfo(client *http.Client) map[string]string {
	info := map[string]string{"server": "Cloudflare", "ip": "Unknown"}

	resp, err := client.Get("https://speed.cloudflare.com/meta")
	if err != nil {
		return info
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if idx := strings.Index(bodyStr, `"colo":"`); idx != -1 {
		start := idx + 8
		if end := strings.Index(bodyStr[start:], `"`); end != -1 {
			info["server"] = "Cloudflare (" + bodyStr[start:start+end] + ")"
		}
	}

	if idx := strings.Index(bodyStr, `"clientIp":"`); idx != -1 {
		start := idx + 12
		if end := strings.Index(bodyStr[start:], `"`); end != -1 {
			info["ip"] = bodyStr[start : start+end]
		}
	}

	return info
}

func buildSpeedtestResult(info map[string]string, ping, jitter, dlSpeed, dlTime float64, dlSize int64, ulSpeed, ulTime float64, ulSize int64) string {
	var sb strings.Builder

	sb.WriteString("<b>Speedtest Results</b>\n\n")

	sb.WriteString("<b>Server</b>\n")
	sb.WriteString(fmt.Sprintf("├ Location : <code>%s</code>\n", info["server"]))
	sb.WriteString(fmt.Sprintf("└ Your IP  : <code>%s</code>\n\n", info["ip"]))

	sb.WriteString("<b>Latency</b>\n")
	sb.WriteString(fmt.Sprintf("├ Ping   : <code>%.2f ms</code>\n", ping))
	sb.WriteString(fmt.Sprintf("└ Jitter : <code>%.2f ms</code>\n\n", jitter))

	sb.WriteString("<b>Download</b>\n")
	sb.WriteString(fmt.Sprintf("├ Speed : <code>%.2f Mbps</code>\n", dlSpeed))
	sb.WriteString(fmt.Sprintf("├ Time  : <code>%.2fs</code>\n", dlTime))
	sb.WriteString(fmt.Sprintf("└ Size  : <code>%.2f MB</code>\n\n", float64(dlSize)/1024/1024))

	sb.WriteString("<b>Upload</b>\n")
	if ulSpeed > 0 {
		sb.WriteString(fmt.Sprintf("├ Speed : <code>%.2f Mbps</code>\n", ulSpeed))
		sb.WriteString(fmt.Sprintf("├ Time  : <code>%.2fs</code>\n", ulTime))
		sb.WriteString(fmt.Sprintf("└ Size  : <code>%.2f MB</code>", float64(ulSize)/1024/1024))
	} else {
		sb.WriteString("└ Status : <code>Failed</code>")
	}

	return sb.String()
}
