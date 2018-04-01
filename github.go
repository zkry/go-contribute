package main

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
)

func waitForRemainingLimit(cl *github.Client, isCore bool, minLimit int) {
	for {
		rateLimits, _, err := cl.RateLimits(context.Background())
		if err != nil {
			log.Printf("could not access rate limit information: %s\n", err)
			<-time.After(time.Second * 1)
			continue
		}

		var rate int
		var limit int
		if isCore {
			rate = rateLimits.GetCore().Remaining
			limit = rateLimits.GetCore().Limit
		} else {
			rate = rateLimits.GetSearch().Remaining
			limit = rateLimits.GetSearch().Limit
		}

		if rate < minLimit {
			log.Printf("Not enough rate limit: %d/%d/%d\n", rate, minLimit, limit)
			<-time.After(time.Second * 60)
			continue
		}
		log.Printf("Rate limit: %d/%d\n", rate, limit)
		break
	}
}

// Determine wheather a github label color's text should be white or black
func colorFromBGColor(bg string) string {
	if len(bg) != 6 {
		return "black"
	}
	c1, err := strconv.ParseUint(bg[0:2], 16, 8)
	if err != nil {
		return "black"
	}
	c2, err := strconv.ParseUint(bg[2:4], 16, 8)
	if err != nil {
		return "black"
	}
	c3, err := strconv.ParseUint(bg[4:6], 16, 8)
	if err != nil {
		return "black"
	}

	if (c1+c2+c3)/3 >= 150 {
		return "black"
	}
	return "white"
}

// isHelpfulLabel returns wheather or not the label is one that welcomes
// a contributor 'helping'. Think of 'help wanted' or 'beginner friendly'
func isHelpfulLabel(l string) bool {
	l = strings.ToLower(l)
	if l == "help wanted" ||
		l == "good first issue" ||
		l == "exp/beginner" ||
		l == "level/beginner" ||
		l == "contribution welcome" ||
		l == "easy" {
		return true
	}
	return false
}
