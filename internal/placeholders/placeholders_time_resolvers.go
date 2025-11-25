package placeholders

import (
	"strconv"
	"time"
)

func resolveUnixTimestamp() (string, error) {
	return strconv.FormatInt(time.Now().UTC().Unix(), 10), nil
}

func resolveISO8601Timestamp() (string, error) {
	return time.Now().UTC().Format(time.RFC3339), nil
}
