package debug

import "fmt"

var messages []string

func AddMessage(format string, a ...any) {
	messages = append([]string{fmt.Sprintf(format, a...)}, messages...)
}

func GetMessages() []string {
	return messages
}
