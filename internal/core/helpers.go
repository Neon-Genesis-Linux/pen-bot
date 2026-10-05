package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

// IsGuild reports whether the interaction that produced e originated in a
// guild. GuildID is nil for direct messages, where guild-scoped commands and
// REST calls (member updates, moderative actions) are unavailable.
func IsGuild[T interface{ GuildID() *snowflake.ID }](e T) bool {
	return e.GuildID() != nil
}

type InteractionResponder interface {
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
	UpdateInteractionResponse(messageUpdate discord.MessageUpdate, opts ...rest.RequestOpt) (*discord.Message, error)
}

// ReplyError replies to an interaction with an ephemeral error message.
// It should be used when the interaction has not been deferred.
func ReplyError(e InteractionResponder, msg string) error {
	return e.CreateMessage(discord.MessageCreate{
		Content: msg,
		Flags:   discord.MessageFlagEphemeral,
	})
}

// EditError edits an existing interaction response with error text and clears embeds/components.
// It should be used when the interaction was already deferred (ephemeral status depends on the original defer).
func EditError(e InteractionResponder, msg string) error {
	_, err := e.UpdateInteractionResponse(discord.MessageUpdate{
		Content:    &msg,
		Embeds:     &[]discord.Embed{},
		Components: &[]discord.LayoutComponent{},
	})
	return err
}

var durationUnits = []struct {
	name string
	secs int64
}{
	{"day", 24 * 3600},
	{"hour", 3600},
	{"minute", 60},
	{"second", 1},
}

// FormatDuration converts a duration into a human readable string (e.g., "1 hour and 30 minutes").
func FormatDuration(duration time.Duration) string {
	seconds := int64(duration.Round(time.Second).Seconds())
	if seconds <= 0 {
		return "0 seconds"
	}

	var parts []string
	for _, u := range durationUnits {
		if seconds >= u.secs {
			count := seconds / u.secs
			seconds %= u.secs

			name := u.name
			if count > 1 {
				name += "s"
			}
			parts = append(parts, fmt.Sprintf("%d %s", count, name))
		}
	}

	switch len(parts) {
	case 0:
		return "0 seconds"
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
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
// are fine: ParseDurationExt sums them, so "1d12h" becomes "24h12h" and still totals 36h.
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

// ParseDurationExt parses a user-supplied duration string, supporting day ("d")
// and week ("w") units in addition to standard Go duration units.
// The resulting duration is rounded to whole seconds.
func ParseDurationExt(s string) (time.Duration, error) {
	d, err := time.ParseDuration(expandLongUnits(s))
	if err != nil {
		return 0, fmt.Errorf("'%s' is not a valid duration", s)
	}

	return d.Round(time.Second), nil
}
