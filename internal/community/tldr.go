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

	// Reject the characters that would change the URL's structure: control
	// characters, space, and the path, query, fragment and escape separators
	// `/`, `?`, `#` and `\`. url.PathEscape escapes the remaining characters
	// that need it; the ones it leaves alone (letters, digits, `-._~` and
	// `$&+=:@`) are all legal in a path segment and cannot start a query or
	// fragment. The length cap bounds the URL as tldr adds pages.
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
		return core.ReplyError(e, "Missing required `command` parameter.")
	}

	command = strings.ToLower(strings.TrimSpace(command))
	command = strings.Join(strings.Fields(command), "-")

	if command == "" {
		return core.ReplyError(e, "Please provide a valid command name.")
	}

	if !tldrCommandPattern.MatchString(command) {
		return core.ReplyError(e, "That command name contains invalid characters.")
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
			desc = "404: There's no one, but us penguins."
		}
		return core.EditError(e, desc)
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
