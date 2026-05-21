package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type TokenType string

const (
	TokenTypeUser TokenType = "user"
	TokenTypeBot  TokenType = "bot"
)

const (
	envUserToken = "SLACK_USER_TOKEN"
	envBotToken  = "SLACK_BOT_TOKEN"
)

var ErrTokenMissing = errors.New("slack token is not set in environment")

func LoadToken(t TokenType) (string, error) {
	var envKey string
	switch t {
	case TokenTypeUser:
		envKey = envUserToken
	case TokenTypeBot:
		envKey = envBotToken
	default:
		return "", fmt.Errorf("unknown token type: %s", t)
	}

	v := os.Getenv(envKey)
	if v == "" {
		return "", fmt.Errorf("loading %s token: %w", t, ErrTokenMissing)
	}
	return v, nil
}

func ParseTokenType(s string) (TokenType, error) {
	switch strings.ToLower(s) {
	case "user":
		return TokenTypeUser, nil
	case "bot":
		return TokenTypeBot, nil
	default:
		return "", fmt.Errorf("unknown token type: %s", s)
	}
}
