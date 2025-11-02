package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type cliCommand struct {
	name        string
	description string
	callback    func() error
}

func cleanInput(text string) []string {
	text = strings.ToLower(text)
	var result []string
	word := ""
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if ch != ' ' {
			word += string(ch)
		} else {
			if word != "" {
				result = append(result, word)
				word = ""
			}
		}
	}
	if word != "" {
		result = append(result, word)
	}
	return result
}

func commandExit() error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp() error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println()
	fmt.Println("help: Displays a help message")
	fmt.Println("exit: Exits the Pokedex")
	return nil
}

func main() {
	commands := map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		input := scanner.Text()

		cleaned := cleanInput(input)
		if len(cleaned) > 0 {
			command := cleaned[0]
			if cmd, ok := commands[command]; ok {
				err := cmd.callback()
				if err != nil {
					fmt.Println(err)
				}
			} else {
				fmt.Println("Unknown command")
			}
		}
	}
}
