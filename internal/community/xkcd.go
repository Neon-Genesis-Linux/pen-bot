package community

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/go-resty/resty/v2"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
)

const xkcdCommand = "xkcd"

var xkcdClient = resty.New().
	SetTimeout(10 * time.Second).
	SetRedirectPolicy(resty.NoRedirectPolicy())

type xkcdMetadata struct {
	Title      string `json:"title"`
	Num        int    `json:"num"`
	Img        string `json:"img"`
	Year       string `json:"year"`
	Month      string `json:"month"`
	Day        string `json:"day"`
	Transcript string `json:"transcript"`
	Alt        string `json:"alt"`
	Link       string `json:"link"`
	News       string `json:"news"`
	SafeTitle  string `json:"safe_title"`
}

func registerXkcdCommands() {
	core.RegisterCommands(
		discord.SlashCommandCreate{
			Name:        xkcdCommand,
			Description: "Fetch an xkcd comic",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:        "get",
					Description: "Fetch a specific comic by number",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionInt{
							Name:        "number",
							Description: "The comic number to fetch",
							Required:    true,
							MinValue:    new(int),
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "random",
					Description: "Fetch a random xkcd comic",
				},
				discord.ApplicationCommandOptionSubCommand{
					Name:        "latest",
					Description: "Fetch the latest xkcd comic",
				},
			},
		},
	)

	h := core.Mux()
	h.Route("/"+xkcdCommand, func(r handler.Router) {
		r.SlashCommand("/get", handleXkcdGet)
		r.SlashCommand("/random", handleXkcdRandom)
		r.SlashCommand("/latest", handleXkcdLatest)
	})
}

func getXkcdMetadata(numStr string) (xkcdMetadata, error) {
	var query string
	if numStr != "" {
		if _, err := strconv.Atoi(numStr); err != nil {
			return xkcdMetadata{}, fmt.Errorf("invalid comic number: %q", numStr)
		}
		query = numStr
	}

	var metadata xkcdMetadata
	resp, err := xkcdClient.R().SetResult(&metadata).Get(fmt.Sprintf("https://xkcd.com/%s/info.0.json", query))
	if err != nil {
		slog.Error("XKCD: Connection error", slog.Any("error", err))
		return xkcdMetadata{}, err
	}
	if !resp.IsSuccess() {
		return xkcdMetadata{}, fmt.Errorf("XKCD: unexpected status %d", resp.StatusCode())
	}
	return metadata, nil
}

func handleXkcdLatest(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	metadata, err := getXkcdMetadata("")
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{Content: "Unable to fetch metadata for current xkcd"})
	}
	return sendXkcd(e, metadata)
}

func handleXkcdGet(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	num, ok := data.OptInt("number")
	if !ok {
		return e.CreateMessage(discord.MessageCreate{Content: "Missing required `number` parameter."})
	}
	metadata, err := getXkcdMetadata(strconv.Itoa(num))
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{Content: fmt.Sprintf("Unable to fetch metadata for xkcd #%d", num)})
	}
	return sendXkcd(e, metadata)
}

func handleXkcdRandom(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	resp, err := xkcdClient.R().Get("https://c.xkcd.com/random/comic/")
	if err != nil {
		slog.Error("XKCD: random fetch error", slog.Any("error", err))
		return e.CreateMessage(discord.MessageCreate{Content: "Unable to fetch random xkcd"})
	}

	if resp.StatusCode() != http.StatusFound {
		slog.Error("XKCD: unexpected status", slog.Int("status", resp.StatusCode()))
		return e.CreateMessage(discord.MessageCreate{Content: "Unexpected response from xkcd"})
	}

	loc := resp.Header().Get("Location")
	if loc == "" {
		return e.CreateMessage(discord.MessageCreate{Content: "No redirect location found"})
	}

	loc = strings.TrimSuffix(loc, "/")
	numStr := loc[strings.LastIndex(loc, "/")+1:]

	if _, err := strconv.Atoi(numStr); err != nil {
		slog.Error("XKCD: failed to parse comic number", slog.String("location", loc))
		return e.CreateMessage(discord.MessageCreate{Content: "Failed to parse redirected comic number"})
	}

	metadata, err := getXkcdMetadata(numStr)
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{Content: fmt.Sprintf("Unable to fetch metadata for xkcd #%s", numStr)})
	}
	return sendXkcd(e, metadata)
}

func sendXkcd(e *handler.CommandEvent, metadata xkcdMetadata) error {
	embed := discord.NewEmbed().
		WithTitle(metadata.Title).
		WithURL(fmt.Sprintf("https://xkcd.com/%d", metadata.Num)).
		WithDescription(metadata.Alt).
		WithImage(metadata.Img).
		WithFooter(fmt.Sprintf("xkcd #%d — %s/%s/%s", metadata.Num, metadata.Month, metadata.Day, metadata.Year), "").
		WithColor(0x96A8C8)
	return e.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{embed}})
}
