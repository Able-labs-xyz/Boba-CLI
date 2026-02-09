package ui

import (
	"math/rand"
)

// Glitch charset — characters used for scrambled text
const glitchChars = "!@#$%^&*()_+-=[]{}|;:<>?/~0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Block glitch charset — heavier characters for logo decryption
const blockGlitchChars = "█▓▒░▄▀▐▌■□▪▫#@$%&*+=~"

// GlitchText returns target text with characters scrambled based on progress (0.0→1.0).
// At 0.0 all non-space characters are random; at 1.0 all are resolved.
// Characters resolve left-to-right with some leading randomness for a "decrypt" feel.
func GlitchText(target string, progress float64) string {
	runes := []rune(target)
	result := make([]rune, len(runes))
	totalNonSpace := 0
	for _, r := range runes {
		if r != ' ' && r != '\n' {
			totalNonSpace++
		}
	}

	resolved := int(float64(totalNonSpace) * progress)
	nonSpaceIdx := 0

	for i, r := range runes {
		if r == ' ' || r == '\n' {
			result[i] = r
			continue
		}

		if nonSpaceIdx < resolved {
			result[i] = r
		} else if nonSpaceIdx < resolved+3 && rand.Float64() < 0.3 {
			// Near the resolve frontier — occasionally show correct char
			result[i] = r
		} else {
			// Scrambled
			result[i] = rune(glitchChars[rand.Intn(len(glitchChars))])
		}
		nonSpaceIdx++
	}
	return string(result)
}

// GlitchTextBlock is like GlitchText but uses block characters for logo-style text.
func GlitchTextBlock(target string, progress float64) string {
	runes := []rune(target)
	result := make([]rune, len(runes))
	totalNonSpace := 0
	for _, r := range runes {
		if r != ' ' && r != '\n' {
			totalNonSpace++
		}
	}

	resolved := int(float64(totalNonSpace) * progress)
	nonSpaceIdx := 0

	for i, r := range runes {
		if r == ' ' || r == '\n' {
			result[i] = r
			continue
		}

		if nonSpaceIdx < resolved {
			result[i] = r
		} else if nonSpaceIdx < resolved+5 && rand.Float64() < 0.4 {
			result[i] = r
		} else {
			chars := []rune(blockGlitchChars)
			result[i] = chars[rand.Intn(len(chars))]
		}
		nonSpaceIdx++
	}
	return string(result)
}

// GlitchLines applies GlitchTextBlock to multiple lines at once.
func GlitchLines(lines []string, progress float64) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		// Each line resolves at a slightly different rate for a wave effect
		lineProgress := progress + float64(i)*0.03
		if lineProgress > 1.0 {
			lineProgress = 1.0
		}
		if lineProgress < 0 {
			lineProgress = 0
		}
		result[i] = GlitchTextBlock(line, lineProgress)
	}
	return result
}

