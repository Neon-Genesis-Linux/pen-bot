package moderation

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/omit"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
)

const (
	selfTimeoutCommand = "self-timeout"

	// Discord allows timeouts up to 28 days; self-timeout is capped well below that.
	// parseSelfTimeoutDuration already understands "d" and "w" terms, so raising this
	// bound is the only change needed to permit week-long self timeouts.
	selfTimeoutLower = 10 * time.Second
	selfTimeoutUpper = 3 * 24 * time.Hour
)

func selfTimeoutCommandEmbed(title string, desc string) discord.Embed {
	return discord.NewEmbed().
		WithTitle(title).
		WithDescription(desc).
		WithColor(0xffc34d)
}

func selfTimeoutCommandErrorEmbed(title string, desc string) discord.Embed {
	return discord.NewEmbed().
		WithTitle(title).
		WithDescription(desc).
		WithColor(0xff0000)
}

// sendEmbed replaces the deferred interaction response with a single embed, optionally
// carrying a row of buttons. Passing a nil buttons leaves the response component-free.
func sendEmbed(e *handler.CommandEvent, embed discord.Embed, buttons *[2]discord.ButtonComponent) error {
	update := discord.MessageUpdate{
		Embeds: &[]discord.Embed{embed},
	}

	// handle buttons
	if buttons != nil {
		row := make([]discord.InteractiveComponent, len(buttons))
		for i, btn := range buttons {
			row[i] = btn
		}

		update.Components = &[]discord.LayoutComponent{
			discord.ActionRowComponent{Components: row},
		}
	}

	_, err := e.UpdateInteractionResponse(update)
	return err
}

// Helper to update the embed, and clear buttons (used in button handlers).
func updateEmbed(e *handler.ComponentEvent, embed discord.Embed) error {
	return e.UpdateMessage(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{embed},
		Components: &[]discord.LayoutComponent{},
	})
}

func handleSelfTimeoutCommandGoBack(_ discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	// overwrite the original message
	embed := selfTimeoutCommandEmbed("Timeout Cancelled", "See you around!")
	return updateEmbed(e, embed)
}

// longTerm matches the week and day units, which time.ParseDuration does not support.
// A duration string is otherwise a sequence of <number><unit> terms, and no unit Go
// does understand contains "w" or "d", so this cannot match a fragment of "ns", "us",
// "ms" or "s". The leading group forces the term to start the string or follow a
// non-numeric character: without it, "1.5.5d" would match the "5.5d" tail and expand
// to "1.132h", which parses as 1.132 hours.
var longTerm = regexp.MustCompile(`(^|[^0-9.])(\d+(?:\.\d+)?)([wd])`)

// hoursPerLongUnit is how many hours each unit expandLongUnits rewrites into. Discord's
// own /timeout takes day and week suffixes, but Go's parser stops at "h".
var hoursPerLongUnit = map[byte]float64{
	'w': 7 * 24,
	'd': 24,
}

// expandLongUnits rewrites week and day terms into hour terms, so that
// time.ParseDuration accepts them ("1d" becomes "24h", "2w" "336h"). Duplicate units
// are fine: ParseDuration sums them, so "1d12h" becomes "24h12h" and still totals 36h.
func expandLongUnits(s string) string {
	// Iterate to a fixpoint. One pass is not always enough: "1w2d" becomes "168h2d" on
	// the first, and the trailing "2d" only becomes matchable once "1w" has stopped
	// occupying the prefix slot the pattern needs. Every match consumes one unit
	// character and emits none, so each pass makes progress and this terminates.
	for replaced := true; replaced; {
		replaced = false
		s = longTerm.ReplaceAllStringFunc(s, func(term string) string {
			replaced = true
			groups := longTerm.FindStringSubmatch(term)
			value, _ := strconv.ParseFloat(groups[2], 64)
			return groups[1] + strconv.FormatFloat(value*hoursPerLongUnit[groups[3][0]], 'f', -1, 64) + "h"
		})
	}
	return s
}

// parseSelfTimeoutDuration turns a user-supplied duration string into the value
// that will actually be applied. It is the single validation point: it runs once,
// on slash command invocation, and the value it returns is the only thing that
// can reach the confirmation button (see registerSelfTimeoutCommand).
func parseSelfTimeoutDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(expandLongUnits(s))
	if err != nil {
		return 0, fmt.Errorf("'%s' is not a valid duration", s)
	}

	// Truncate millisecond/nanosecond precision, so that Duration.String() — which is
	// what the confirmation prompt and the button custom ID both render — is an exact
	// description of the timeout that gets applied.
	d = d.Round(time.Second)

	if d < selfTimeoutLower {
		return 0, fmt.Errorf("'%s' is shorter than the %g second minimum", s, selfTimeoutLower.Seconds())
	}

	if d > selfTimeoutUpper {
		return 0, fmt.Errorf("'%s' is longer than the %g hour maximum", s, selfTimeoutUpper.Hours())
	}

	return d, nil
}

func handleSelfTimeoutCommandConfirm(_ discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	duration, err := time.ParseDuration(e.Vars["duration"])
	if err != nil {
		// NOTE: unreachable via the button; kept as the guard against a custom ID that
		// did not come from parseSelfTimeoutDuration.
		embed := selfTimeoutCommandErrorEmbed("Invalid duration!", "The specified duration is invalid!\n"+
			"This error should have been caught earlier...")
		return updateEmbed(e, embed)
	}

	if !core.IsGuild(e) {
		// NOTE: should have been caught earlier
		embed := selfTimeoutCommandErrorEmbed("Command unavailable!", "This command can only be used in servers!")
		return updateEmbed(e, embed)
	}

	// No range check here: the custom ID carries a duration that parseSelfTimeoutDuration
	// already accepted, and Duration.String() round-trips exactly, so re-deriving the
	// bounds could only ever disagree with what the user confirmed.

	// actually time out the user
	until := time.Now().Add(duration)
	GuildID := e.GuildID()
	userID := e.Member().User.ID
	_, err = e.Client().Rest.UpdateMember(*GuildID, userID, discord.MemberUpdate{
		CommunicationDisabledUntil: omit.New(&until),
	}, rest.WithReason("self-timeout"))

	if err != nil {
		slog.Error("self-timeout: member update failed",
			slog.String("guild", GuildID.String()),
			slog.String("user", userID.String()),
			slog.Any("error", err),
		)

		desc := fmt.Sprintf("The bot was unable to timeout <@%s>.\nPlease try again later.", userID)

		// A 403 means Discord refused: the bot lacks Moderate Members, or its highest
		// role sits below the member's. Match on the status, not the body — the error code
		// here is 50013 ("Missing Permissions"), which Discord may omit entirely, and
		// 50013 is client-defined so its meaning is not stable across endpoints.
		var restErr *rest.Error
		if errors.As(err, &restErr) && restErr.Response != nil && restErr.Response.StatusCode == http.StatusForbidden {
			desc = fmt.Sprintf("The bot was unable to timeout <@%s>.\n"+
				"It needs the *Time out members* permission and a role above yours.", userID)
		}

		embed := selfTimeoutCommandErrorEmbed("Unable to timeout user!", desc)
		return updateEmbed(e, embed)
	}

	// TODO: log the self-timeout in a configured staff channel once logging feature is added

	embed := selfTimeoutCommandEmbed("Successfully timed out", "See you *later*!")
	return updateEmbed(e, embed)
}

func handleSelfTimeoutCommand(interaction discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if err := e.DeferCreateMessage(true); err != nil {
		return err
	}

	if !core.IsGuild(e) {
		return sendEmbed(e, selfTimeoutCommandErrorEmbed("Command unavailable!", "This command can only be used in servers!"), nil)
	}

	durationString, ok := interaction.OptString("duration")
	if !ok {
		return sendEmbed(e, selfTimeoutCommandErrorEmbed("Unset duration", "Duration was not set!"), nil)
	}

	duration, err := parseSelfTimeoutDuration(durationString)
	if err != nil {
		return sendEmbed(e, selfTimeoutCommandErrorEmbed("Invalid duration", err.Error()), nil)
	}

	// One value for the prompt and the custom ID, so what the user confirms is exactly
	// what gets applied. Duration.String() is canonical and round-trips exactly.
	effective := duration.String()

	embed := selfTimeoutCommandEmbed("Confirm timeout", fmt.Sprintf(
		"You are about to time yourself out, **this action is irreversible**, "+
			"there is *no provision* for this to be undone by moderation!"+
			"\nConfirm `%s` timeout?",
		effective))

	buttons := [2]discord.ButtonComponent{
		discord.NewDangerButton("I understand", "/"+selfTimeoutCommand+"/Confirm/"+effective),
		discord.NewSecondaryButton("Cancel", "/"+selfTimeoutCommand+"/GoBack/"),
	}

	return sendEmbed(e, embed, &buttons)
}

func registerSelfTimeoutCommand() {
	core.RegisterCommands(
		discord.SlashCommandCreate{
			Name:        selfTimeoutCommand,
			Description: "Time yourself out for a set period of time.",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name: "duration",
					// Discord caps option descriptions at 100 characters and silently
					// truncates past that, so keep this short if the wording changes.
					Description: fmt.Sprintf("How long the self timeout should last (e.g. 30m, 1h10m, 1d, 1w)\n"+
						"Accepted range: [%s, %gh]", selfTimeoutLower, selfTimeoutUpper.Hours()),
					Required: true,
				},
			},
		},
	)

	// {duration} is filled from a value parseSelfTimeoutDuration already accepted, and
	// Duration.String() emits only [0-9hmsµun.] — no '/', so the segment cannot escape
	// into another route. handleSelfTimeoutCommandConfirm therefore trusts it and does
	// not re-check the bounds.
	h := core.Mux()
	h.ButtonComponent("/"+selfTimeoutCommand+"/Confirm/{duration}", handleSelfTimeoutCommandConfirm)
	h.ButtonComponent("/"+selfTimeoutCommand+"/GoBack/", handleSelfTimeoutCommandGoBack)
	h.SlashCommand("/"+selfTimeoutCommand, handleSelfTimeoutCommand)
}
