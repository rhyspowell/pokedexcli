package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rhyspowell/pokedexcli/internal/pokecache"
)

type config struct {
	Next     *string
	Previous *string
	Cache    *pokecache.Cache
}

type cliCommand struct {
	name        string
	description string
	callback    func(*config) error
}

type locationAreaResponse struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
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

func commandExit(cfg *config) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *config) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println()
	fmt.Println("help: Displays a help message")
	fmt.Println("exit: Exits the Pokedex")
	fmt.Println("map: Displays the names of 20 location areas")
	fmt.Println("mapb: Displays the previous 20 location areas")
	return nil
}

func fetchLocationAreas(cfg *config, url string) (locationAreaResponse, error) {
	var result locationAreaResponse

	// Check cache first
	if body, ok := cfg.Cache.Get(url); ok {
		if err := json.Unmarshal(body, &result); err != nil {
			return result, fmt.Errorf("error parsing cached JSON: %v", err)
		}
		return result, nil
	}

	// Cache miss - make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return result, fmt.Errorf("error fetching location areas: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("error reading response: %v", err)
	}

	// Add to cache
	cfg.Cache.Add(url, body)

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("error parsing JSON: %v", err)
	}

	return result, nil
}

func commandMap(cfg *config) error {
	var url string
	if cfg.Next != nil && *cfg.Next != "" {
		url = *cfg.Next
	} else {
		url = "https://pokeapi.co/api/v2/location-area"
	}

	result, err := fetchLocationAreas(cfg, url)
	if err != nil {
		return err
	}

	cfg.Next = result.Next
	cfg.Previous = result.Previous

	for _, area := range result.Results {
		fmt.Println(area.Name)
	}

	return nil
}

func commandMapb(cfg *config) error {
	if cfg.Previous == nil || *cfg.Previous == "" {
		fmt.Println("you're on the first page")
		return nil
	}

	url := *cfg.Previous

	result, err := fetchLocationAreas(cfg, url)
	if err != nil {
		return err
	}

	cfg.Next = result.Next
	cfg.Previous = result.Previous

	for _, area := range result.Results {
		fmt.Println(area.Name)
	}

	return nil
}

func main() {
	cfg := &config{
		Cache: pokecache.NewCache(5 * time.Second),
	}

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
		"map": {
			name:        "map",
			description: "Displays the names of 20 location areas",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Displays the previous 20 location areas",
			callback:    commandMapb,
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
				err := cmd.callback(cfg)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				fmt.Println("Unknown command")
			}
		}
	}
}
