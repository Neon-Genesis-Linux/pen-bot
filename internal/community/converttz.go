package community

import (
	"errors"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Neon-Genesis-Linux/pen-bot/internal/core"
)

const (
	convertTzCommand    = "converttz"
	toTimezoneArgName   = "to-timezone"
	fromTimezoneArgName = "from-timezone"
)

func registerTimezoneCommands() {
	core.RegisterCommands(
		discord.SlashCommandCreate{
			Name:        convertTzCommand,
			Description: "Convert a time from one timezone to another",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "time",
					Description: "The time to convert",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        fromTimezoneArgName,
					Description: "The timezone 'time' is in",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        toTimezoneArgName,
					Description: "The timezone you want 'time' to be in, uses Discord's timestamp feature if unset",
					Required:    false,
				},
			},
		},
	)

	h := core.Mux()
	h.SlashCommand("/"+convertTzCommand, handleConvertTzCommand)
}

// convertTimeLayouts are the input formats accepted by /converttz. The order
// matters: the first layout that parses the value also formats the result.
var convertTimeLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"15:04:05",
	"15:04",
}

// errUnrecognizedTime means no accepted layout parsed the supplied value.
var errUnrecognizedTime = errors.New("unrecognized time format")

// parseConvertTime parses value in fromLoc and returns the matched layout.
//
// Go represents a time parsed from a layout without a date with year zero and
// the date January 1. Such a value is filled in with now's date, in fromLoc,
// so that the time of day keeps the offset in force on that date rather than
// the offset in force on January 1.
func parseConvertTime(value string, fromLoc *time.Location, now time.Time) (time.Time, string, error) {
	for _, layout := range convertTimeLayouts {
		parsed, err := time.ParseInLocation(layout, value, fromLoc)
		if err != nil {
			continue
		}
		if parsed.Year() == 0 {
			parsed = time.Date(
				now.Year(),
				now.Month(),
				now.Day(),
				parsed.Hour(),
				parsed.Minute(),
				parsed.Second(),
				parsed.Nanosecond(),
				fromLoc,
			)
		}
		return parsed, layout, nil
	}
	return time.Time{}, "", errUnrecognizedTime
}

func handleConvertTzCommand(interaction discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if err := e.DeferCreateMessage(true); err != nil {
		return err
	}

	timeStamp, ok := interaction.OptString("time")
	if !ok {
		return core.EditError(e, "Missing required `time` argument!")
	}

	fromTz, ok := interaction.OptString(fromTimezoneArgName)
	if !ok {
		return core.EditError(e, "Missing required `"+fromTimezoneArgName+"` argument!")
	}

	toTz, toTzPresent := interaction.OptString(toTimezoneArgName)

	fromLoc, err := time.LoadLocation(fromTz)
	if err != nil {
		return core.EditError(e, "Invalid `"+fromTimezoneArgName+"`: "+err.Error())
	}

	parsedTime, layout, err := parseConvertTime(timeStamp, fromLoc, time.Now().In(fromLoc))
	if err != nil {
		return core.EditError(e, "Time is in an unrecognized format!")
	}

	if !toTzPresent {
		// Discord renders a timestamp in the viewer's own timezone, so no
		// target timezone has to be chosen for this reply.
		discordFormattedTimestamp := fmt.Sprintf("<t:%d:f>", parsedTime.Unix())

		_, err := e.UpdateInteractionResponse(
			discord.MessageUpdate{
				Content: &discordFormattedTimestamp,
			})
		return err
	}

	toLoc, err := time.LoadLocation(toTz)
	if err != nil {
		return core.EditError(e, "Invalid `"+toTimezoneArgName+"`: "+err.Error())
	}

	result := parsedTime.In(toLoc).Format(layout)
	_, err = e.UpdateInteractionResponse(
		discord.MessageUpdate{
			Content: &result,
		})
	return err
}
