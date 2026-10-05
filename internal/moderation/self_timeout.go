package moderation

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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
	// ParseDurationExt already understands "d" and "w" terms, so raising this
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

func handleSelfTimeoutCommandConfirm(_ discord.ButtonInteractionData, e *handler.ComponentEvent) error {
	if err := e.DeferUpdateMessage(); err != nil {
		return err
	}

	duration, err := time.ParseDuration(e.Vars["duration"])
	if err != nil {
		// NOTE: unreachable via the button; kept as the guard against a custom ID that
		// did not come from handleSelfTimeoutCommand.
		return core.EditError(e, "The specified duration is invalid!")
	}

	if !core.IsGuild(e) {
		// NOTE: should have been caught earlier
		return core.EditError(e, "This command can only be used in servers!")
	}

	// No range check here: the custom ID carries a duration that handleSelfTimeoutCommand
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

		desc := "The bot was unable to timeout. Please try again later."

		// A 403 means Discord refused: the bot lacks Moderate Members, or its highest
		// role sits below the member's. Match on the status, not the body — the error code
		// here is 50013 ("Missing Permissions"), which Discord may omit entirely, and
		// 50013 is client-defined so its meaning is not stable across endpoints.
		var restErr *rest.Error
		if errors.As(err, &restErr) && restErr.Response != nil && restErr.Response.StatusCode == http.StatusForbidden {
			desc = "The bot was unable to timeout.\nIt needs the *Time out members* permission and a role above yours."
		}

		return core.EditError(e, desc)
	}

	// TODO: log the self-timeout in a configured staff channel once logging feature is added

	embed := selfTimeoutCommandEmbed("Successfully timed out", "See you *later*!")
	_, editErr := e.UpdateInteractionResponse(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{embed},
		Components: &[]discord.LayoutComponent{},
	})
	return editErr
}

func handleSelfTimeoutCommand(interaction discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if err := e.DeferCreateMessage(true); err != nil {
		return err
	}

	if !core.IsGuild(e) {
		return core.EditError(e, "This command can only be used in servers!")
	}

	durationString, ok := interaction.OptString("duration")
	if !ok {
		return core.EditError(e, "Duration was not set!")
	}

	duration, err := core.ParseDurationExt(durationString)
	if err != nil {
		return core.EditError(e, err.Error())
	}

	if duration < selfTimeoutLower {
		return core.EditError(e, fmt.Sprintf("%s is shorter than the minimum of %s", core.FormatDuration(duration), core.FormatDuration(selfTimeoutLower)))
	}
	if duration > selfTimeoutUpper {
		return core.EditError(e, fmt.Sprintf("%s is longer than the maximum of %s", core.FormatDuration(duration), core.FormatDuration(selfTimeoutUpper)))
	}

	effective := duration.String()

	embed := selfTimeoutCommandEmbed("Confirm timeout", fmt.Sprintf(
		"You are about to time yourself out, **this action is irreversible**, "+
			"there is *no provision* for this to be undone by moderation!"+
			"\nConfirm **%s** timeout?",
		core.FormatDuration(duration)))

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
					Description: fmt.Sprintf("How long the self timeout should last (e.g. 30m, 1h10m, 1d) Accepted range: [%s, %gh]", selfTimeoutLower, selfTimeoutUpper.Hours()),
					Required:    true,
				},
			},
		},
	)

	// {duration} is filled from a value ParseDurationExt already accepted, and
	// Duration.String() emits only [0-9hmsµun.] — no '/', so the segment cannot escape
	// into another route. handleSelfTimeoutCommandConfirm therefore trusts it and does
	// not re-check the bounds.
	h := core.Mux()
	h.ButtonComponent("/"+selfTimeoutCommand+"/Confirm/{duration}", handleSelfTimeoutCommandConfirm)
	h.ButtonComponent("/"+selfTimeoutCommand+"/GoBack/", handleSelfTimeoutCommandGoBack)
	h.SlashCommand("/"+selfTimeoutCommand, handleSelfTimeoutCommand)
}
