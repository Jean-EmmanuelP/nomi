package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nullswan/nomi/internal/chat"
	"github.com/nullswan/nomi/internal/completion"
	baseprovider "github.com/nullswan/nomi/internal/providers/base"
	"github.com/nullswan/nomi/internal/term"
)

// GenerateCompletion generates a completion using the provided backend.
func GenerateCompletion(
	ctx context.Context,
	conversation chat.Conversation,
	renderer *term.Renderer,
	textToTextBackend baseprovider.TextToTextProvider,
) (string, error) {
	outCh := make(chan completion.Completion)

	go func() {
		defer close(outCh)
		if err := textToTextBackend.GenerateCompletion(ctx, conversation.GetMessages(), outCh); err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				return
			}
			fmt.Printf("Error generating completion: %v\n", err)
		}
	}()

	sb := term.NewScreenBuf(os.Stdout)
	var fullContent string
	currentLine := ""

	for {
		select {
		case cmpl, ok := <-outCh:
			if completion.IsTombStone(cmpl) {
				sb.Clear()

				mdContent, err := renderer.Render(fullContent)
				if err != nil {
					fmt.Println("Error rendering markdown:", err)
					return fullContent, fmt.Errorf(
						"rendering markdown: %w",
						err,
					)
				}

				mdContent = strings.TrimSpace(mdContent)

				sb.WriteLine(mdContent)
				return fullContent, nil
			}

			if !ok {
				fmt.Println()
				return fullContent, errors.New("error reading completion")
			}

			if cmpl.Content() == "" {
				continue
			}

			fullContent += cmpl.Content()
			currentLine += cmpl.Content()
			if strings.Contains(currentLine, "\n") {
				lines := strings.Split(currentLine, "\n")
				for i, line := range lines {
					if i == len(lines)-1 {
						currentLine = line
						continue
					}
					sb.WriteLine(line)
				}
				currentLine = currentLine[strings.LastIndex(currentLine, "\n")+1:]
			}
		case <-ctx.Done():
			return fullContent, errors.New("context canceled")
		}
	}
}
