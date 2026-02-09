package ui

import (
	"math/rand"
	"strings"
)

// Matrix rain charset — techy ASCII characters
const matrixChars = "0123456789ABCDEFabcdef:.<>{}[]|/\\+-*#@$!?~^&="

// matrixColumn tracks a single falling stream.
type matrixColumn struct {
	pos    int    // current head row (negative = delayed start)
	speed  int    // ticks between advances (1=fast, 3=slow)
	length int    // trail length
	chars  []byte // character at each row
	tick   int    // counter for speed gating
}

// MatrixRain manages a grid of falling characters.
type MatrixRain struct {
	Width  int
	Height int
	cols   []matrixColumn
	Frame  int
}

// NewMatrixRain creates a matrix rain effect for the given dimensions.
func NewMatrixRain(width, height int) *MatrixRain {
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}

	cols := make([]matrixColumn, width)
	for i := range cols {
		cols[i] = matrixColumn{
			pos:    -(rand.Intn(height + 10)),
			speed:  1 + rand.Intn(3),
			length: 4 + rand.Intn(height/2+1),
			chars:  make([]byte, height),
		}
		for j := range cols[i].chars {
			cols[i].chars[j] = matrixChars[rand.Intn(len(matrixChars))]
		}
	}

	return &MatrixRain{Width: width, Height: height, cols: cols}
}

// Tick advances the rain by one frame.
func (m *MatrixRain) Tick() {
	m.Frame++
	for i := range m.cols {
		c := &m.cols[i]
		c.tick++
		if c.tick < c.speed {
			continue
		}
		c.tick = 0
		c.pos++

		// Write new char at head
		if c.pos >= 0 && c.pos < m.Height {
			c.chars[c.pos] = matrixChars[rand.Intn(len(matrixChars))]
		}

		// Shimmer: randomly change trail characters
		for j := max(0, c.pos-c.length); j < c.pos && j < m.Height; j++ {
			if j >= 0 && rand.Float64() < 0.08 {
				c.chars[j] = matrixChars[rand.Intn(len(matrixChars))]
			}
		}

		// Reset when fully past screen
		if c.pos-c.length > m.Height {
			c.pos = -(rand.Intn(m.Height/2 + 5))
			c.speed = 1 + rand.Intn(3)
			c.length = 4 + rand.Intn(m.Height/2+1)
		}
	}
}

// Render returns the current frame as a string with ANSI color codes.
// Uses raw ANSI for performance (renders ~2400 chars per frame at 30fps).
func (m *MatrixRain) Render() string {
	var b strings.Builder
	b.Grow(m.Width * m.Height * 8)

	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			c := &m.cols[x]

			// Not in trail range → empty
			if c.pos < 0 || y > c.pos || y < c.pos-c.length {
				b.WriteByte(' ')
				continue
			}

			dist := c.pos - y
			ch := c.chars[y]

			switch {
			case dist == 0:
				// Head: bright white bold
				b.WriteString("\033[1;97m")
				b.WriteByte(ch)
				b.WriteString("\033[0m")
			case dist == 1:
				// Near head: bright green bold
				b.WriteString("\033[1;92m")
				b.WriteByte(ch)
				b.WriteString("\033[0m")
			case dist <= c.length/4:
				b.WriteString("\033[92m")
				b.WriteByte(ch)
				b.WriteString("\033[0m")
			case dist <= c.length/2:
				b.WriteString("\033[32m")
				b.WriteByte(ch)
				b.WriteString("\033[0m")
			case dist <= c.length*3/4:
				b.WriteString("\033[2;32m")
				b.WriteByte(ch)
				b.WriteString("\033[0m")
			default:
				b.WriteString("\033[38;5;22m")
				b.WriteByte(ch)
				b.WriteString("\033[0m")
			}
		}
		if y < m.Height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// LogoLines returns the raw logo text for overlay composition.
func LogoLines() []string {
	return logoLines
}
