package community

import (
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

func handleConvertTzCommand(interaction discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if err := e.DeferCreateMessage(true); err != nil {
		return err
	}

	// get argument 'time'
	timeStamp, ok := interaction.OptString("time")
	if !ok {
		return core.EditError(e, "Missing required `time` argument!")
	}

	// same as above but with fromTz
	fromTz, ok := interaction.OptString(fromTimezoneArgName)
	if !ok {
		return core.EditError(e, "Missing required `"+fromTimezoneArgName+"` argument!")
	}

	// if toTzPresent is false, we do something different later...
	toTz, toTzPresent := interaction.OptString(toTimezoneArgName)

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"15:04:05",
		"15:04",
	}

	var parsedTime time.Time
	layout := ""

	// convert fromTz to location...
	fromLoc, err := time.LoadLocation(fromTz)
	if err != nil {
		return core.EditError(e, "Invalid `"+fromTimezoneArgName+"`: "+err.Error())
	}

	for _, candidate := range layouts {
		var err error // if I use := then go will throw an error with 'parsedTime'

		parsedTime, err = time.ParseInLocation(candidate, timeStamp, fromLoc)
		if err == nil {
			layout = candidate
			break
		}
	}

	// if nothing was matched layout will be ""
	if layout == "" {
		return core.EditError(e, "Time is in an unrecognized format!")
	}

	if parsedTime.Year() == 0 {
		now := time.Now().In(fromLoc)
		parsedTime = time.Date(
			now.Year(),
			parsedTime.Month(),
			parsedTime.Day(),
			parsedTime.Hour(),
			parsedTime.Minute(),
			parsedTime.Second(),
			parsedTime.Nanosecond(),
			fromLoc,
		)
	}

	if !toTzPresent {
		// this will format it, with Discord, like Month, Xnd, year, X:XX (PM/AM)
		discordFormattedTimestamp := fmt.Sprintf("<t:%d:f>", parsedTime.Unix())

		// update original message
		_, err := e.UpdateInteractionResponse(
			discord.MessageUpdate{
				Content: &discordFormattedTimestamp,
			})
		return err
	} // else:

	toLoc, err := time.LoadLocation(toTz)
	if err != nil {
		return core.EditError(e, "Invalid `"+toTimezoneArgName+"`: "+err.Error())
	}

	// format result and update response
	result := parsedTime.In(toLoc).Format(layout)
	_, err = e.UpdateInteractionResponse(
		discord.MessageUpdate{
			Content: &result,
		})
	return err
}
