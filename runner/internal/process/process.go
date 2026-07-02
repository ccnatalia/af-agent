package process

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const MinKeywordLength = 3

type Info struct {
	PID     int
	Command string
}

type Matched struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
}

func NormalizeKeyword(keyword string) (string, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return "", errors.New("keyword is required")
	}
	if len(keyword) < MinKeywordLength {
		return "", fmt.Errorf("keyword must be at least %d characters", MinKeywordLength)
	}
	if strings.ContainsAny(keyword, "\x00\r\n") {
		return "", errors.New("keyword must not contain control characters")
	}

	return keyword, nil
}

func FindMatching(
	keyword string,
	currentPID int,
	list func() ([]Info, error),
) ([]Matched, error) {
	processes, err := list()
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	var matched []Matched
	for _, process := range processes {
		if process.PID <= 0 || process.PID == currentPID {
			continue
		}
		if !strings.Contains(process.Command, keyword) {
			continue
		}

		matched = append(matched, Matched{
			PID:     process.PID,
			Command: process.Command,
		})
	}

	return matched, nil
}

func FindMatchingCurrent(keyword string) ([]Matched, error) {
	return FindMatching(keyword, os.Getpid(), List)
}
