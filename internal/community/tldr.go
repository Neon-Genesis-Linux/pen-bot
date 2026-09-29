package community

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/go-resty/resty/v2"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
)

var tldrClient = resty.New().SetTimeout(10 * time.Second).
	SetBaseURL("https://raw.githubusercontent.com/tldr-pages/tldr/refs/heads/main/pages")

var (
	errTldrNotFound = errors.New("page not found")

	// Reject only characters that make the URL ambiguous or can't appear in a real
	// command name. Everything structural is neutralised by url.PathEscape, whose
	// pass-through set is limited to unreserved path characters, so this stays
	// correct as tldr adds pages. The length cap keeps us under GitHub's URL limit.
	tldrCommandPattern = regexp.MustCompile(`^[^\x00-\x20\x7f/?#\\]{1,64}$`)
)

const tldrCommand = "tldr"

func tldrEmbed(title, desc string) discord.Embed {
	return discord.NewEmbed().
		WithTitle(title).
		WithDescription(desc).
		WithColor(0x7cbcb4).
		WithFooter("tldr-pages", "https://tldr.sh/assets/img/icon.png")
}

func tldrErrorEmbed(desc string) discord.Embed {
	return tldrEmbed("Whoops!", desc)
}

func registerTldrCommands() {
	core.RegisterCommands(
		discord.SlashCommandCreate{
			Name:        tldrCommand,
			Description: "Get a TLDR of a command",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "command",
					Description: "The command (e.g. ls, git commit) cheat sheet to search for",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "platform",
					Description: "Target operating system platform (defaults to common)",
					Required:    false,
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{
							Name:  "Android",
							Value: "android",
						},
						{
							Name:  "Cisco IOS",
							Value: "cisco-ios",
						},
						{
							Name:  "Common (Cross-platform)",
							Value: "common",
						},
						{
							Name:  "DOS",
							Value: "dos",
						},
						{
							Name:  "FreeBSD",
							Value: "freebsd",
						},
						{
							Name:  "Linux",
							Value: "linux",
						},
						{
							Name:  "NetBSD",
							Value: "netbsd",
						},
						{
							Name:  "OpenBSD",
							Value: "openbsd",
						},
						{
							Name:  "macOS (OSX)",
							Value: "osx",
						},
						{
							Name:  "SunOS",
							Value: "sunos",
						},
						{
							Name:  "Windows",
							Value: "windows",
						},
					},
				},
			},
		},
	)

	h := core.Mux()
	h.SlashCommand("/"+tldrCommand, handleTldr)
}

func getTldrContent(platform, command string) (string, error) {
	resp, err := tldrClient.R().Get(fmt.Sprintf("/%s/%s.md", url.PathEscape(platform), url.PathEscape(command)))
	if err != nil {
		slog.Error("TLDR: request failed", slog.String("command", command), slog.Any("error", err))
		return "", err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return "", errTldrNotFound
	}

	if resp.IsError() {
		slog.Error("TLDR: unexpected status", slog.Int("status", resp.StatusCode()), slog.String("command", command))
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode())
	}

	return resp.String(), nil
}

func handleTldr(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	command, ok := data.OptString("command")
	if !ok {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{tldrErrorEmbed("Missing required `command` parameter.")},
		})
	}

	command = strings.ToLower(strings.TrimSpace(command))
	command = strings.Join(strings.Fields(command), "-")

	if command == "" {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{tldrErrorEmbed("Please provide a valid command name.")},
		})
	}

	if !tldrCommandPattern.MatchString(command) {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{tldrErrorEmbed("That command name contains invalid characters.")},
		})
	}

	platform, ok := data.OptString("platform")
	if !ok {
		platform = "common"
	}

	err := e.DeferCreateMessage(false)
	if err != nil {
		return err
	}

	content, err := getTldrContent(platform, command)

	if err != nil {
		desc := "An error occurred while fetching the cheat sheet."
		if errors.Is(err, errTldrNotFound) {
			desc = "404: There's no one, but us chickens."
		}

		_, err := e.UpdateInteractionResponse(discord.MessageUpdate{
			Embeds: &[]discord.Embed{tldrErrorEmbed(desc)},
		})

		return err
	}

	return sendTldr(e, command, platform, content)
}

func sendTldr(e *handler.CommandEvent, command, platform, rawMarkdown string) error {
	lines := strings.Split(rawMarkdown, "\n")
	var result strings.Builder

	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.ReplaceAll(line, "{{", "{")
		line = strings.ReplaceAll(line, "}}", "}")

		result.WriteString(line)
		result.WriteString("\n")
	}

	embed := tldrEmbed(
		fmt.Sprintf("%s (%s)", command, platform),
		strings.TrimSpace(result.String()),
	).WithURL(fmt.Sprintf(
		"https://github.com/tldr-pages/tldr/blob/main/pages/%s/%s.md",
		url.PathEscape(platform), url.PathEscape(command),
	))

	_, err := e.UpdateInteractionResponse(discord.MessageUpdate{
		Embeds: &[]discord.Embed{embed},
	})

	return err
}
