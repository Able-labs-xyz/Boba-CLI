package ui

import (
	"fmt"
	"strings"
)

// StatusBox renders a bordered box with a title and content lines.
func StatusBox(title string, lines []string) string {
	titleRendered := TitleStyle.Render(title)
	content := titleRendered + "\n\n" + strings.Join(lines, "\n")
	return BoxBorder.Render(content)
}

// SuccessBox renders a green-bordered box with a check mark and message.
func SuccessBox(msg string) string {
	icon := SuccessStyle.Render("\u2714") // check mark
	content := fmt.Sprintf("%s  %s", icon, msg)
	return SuccessBoxBorder.Render(content)
}

// ErrorBox renders a red-bordered box with an X mark and message.
func ErrorBox(msg string) string {
	icon := ErrorStyle.Render("\u2718") // X mark
	content := fmt.Sprintf("%s  %s", icon, msg)
	return ErrorBoxBorder.Render(content)
}

