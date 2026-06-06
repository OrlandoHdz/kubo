package services

import (
    "fmt"
    "math/big"
    "strconv"
    "strings"
    "time"

    "github.com/jackc/pgx/v5/pgtype"
)

// toString returns a plain Go string from a DBF value.
func toString(v interface{}) string {
    if v == nil {
        return ""
    }
    return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// toPGText returns pgtype.Text suitable for sqlc generated fields.
func toPGText(v interface{}) pgtype.Text {
    if v == nil {
        return pgtype.Text{Valid: false}
    }
    str := strings.TrimSpace(fmt.Sprintf("%v", v))
    return pgtype.Text{String: str, Valid: true}
}

// toNumeric parses numeric DBF values into pgtype.Numeric.
func toNumeric(v interface{}) pgtype.Numeric {
    if v == nil {
        return pgtype.Numeric{Valid: false}
    }
    var f float64
    switch val := v.(type) {
    case float64:
        f = val
    case int32:
        f = float64(val)
    case int64:
        f = float64(val)
    case int:
        f = float64(val)
    case string:
        f, _ = strconv.ParseFloat(strings.TrimSpace(val), 64)
    default:
        return pgtype.Numeric{Valid: false}
    }
    num := pgtype.Numeric{}
    // Store with two decimal places (scale -2)
    num.Int = big.NewInt(int64(f * 100))
    num.Exp = -2
    num.Valid = true
    return num
}

// toTimestamp converts DBF date/time values into pgtype.Timestamp.
func toTimestamp(v interface{}) pgtype.Timestamp {
    if v == nil {
        return pgtype.Timestamp{Valid: false}
    }
    switch t := v.(type) {
    case time.Time:
        // Zero out time part to UTC midnight
        utc := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
        return pgtype.Timestamp{Time: utc, Valid: true}
    case string:
        str := strings.TrimSpace(t)
        if str == "" {
            return pgtype.Timestamp{Valid: false}
        }
        layouts := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00", "2006-01-02"}
        for _, l := range layouts {
            if parsed, err := time.Parse(l, str); err == nil {
                return pgtype.Timestamp{Time: parsed, Valid: true}
            }
        }
        return pgtype.Timestamp{Valid: false}
    default:
        return pgtype.Timestamp{Valid: false}
    }
}

// parseInt64 parses integer DBF values safely.
func parseInt64(v interface{}) int64 {
    if v == nil {
        return 0
    }
    switch val := v.(type) {
    case int64:
        return val
    case int:
        return int64(val)
    case string:
        i, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
        if err != nil {
            return 0
        }
        return i
    default:
        return 0
    }
}
