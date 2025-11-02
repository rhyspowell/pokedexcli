package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/rhyspowell/pokedexcli/internal/pokecache"
)

type config struct {
	Next     *string
	Previous *string
	Cache    *pokecache.Cache
	Pokedex  map[string]Pokemon
}

type Pokemon struct {
	Name           string `json:"name"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	BaseExperience int    `json:"base_experience"`
	Stats          []struct {
		BaseStat int `json:"base_stat"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"types"`
}

type cliCommand struct {
	name        string
	description string
	callback    func(*config, ...string) error
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

func commandExit(cfg *config, args ...string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *config, args ...string) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println()
	fmt.Println("help: Displays a help message")
	fmt.Println("exit: Exits the Pokedex")
	fmt.Println("map: Displays the names of 20 location areas")
	fmt.Println("mapb: Displays the previous 20 location areas")
	fmt.Println("explore <location-area>: Lists the Pokemon in a location area")
	fmt.Println("catch <pokemon>: Attempt to catch a Pokemon")
	fmt.Println("inspect <pokemon>: Display information about a caught Pokemon")
	fmt.Println("pokedex: Lists all caught Pokemon")
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

func commandMap(cfg *config, args ...string) error {
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

func commandMapb(cfg *config, args ...string) error {
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

type locationAreaDetailResponse struct {
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

func fetchLocationAreaDetail(cfg *config, locationAreaName string) (locationAreaDetailResponse, error) {
	var result locationAreaDetailResponse
	url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s", locationAreaName)

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
		return result, fmt.Errorf("error fetching location area: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return result, fmt.Errorf("location area not found")
	}

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

func commandExplore(cfg *config, args ...string) error {
	if len(args) == 0 {
		fmt.Println("Usage: explore <location-area>")
		return nil
	}

	locationAreaName := args[0]

	result, err := fetchLocationAreaDetail(cfg, locationAreaName)
	if err != nil {
		return err
	}

	if len(result.PokemonEncounters) == 0 {
		fmt.Printf("No Pokemon found in %s\n", locationAreaName)
		return nil
	}

	fmt.Printf("Exploring %s...\n", locationAreaName)
	fmt.Println("Found Pokemon:")
	for _, encounter := range result.PokemonEncounters {
		fmt.Printf("  - %s\n", encounter.Pokemon.Name)
	}

	return nil
}

func fetchPokemon(cfg *config, pokemonName string) (Pokemon, error) {
	var pokemon Pokemon
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", pokemonName)

	// Check cache first
	if body, ok := cfg.Cache.Get(url); ok {
		if err := json.Unmarshal(body, &pokemon); err != nil {
			return pokemon, fmt.Errorf("error parsing cached JSON: %v", err)
		}
		return pokemon, nil
	}

	// Cache miss - make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return pokemon, fmt.Errorf("error fetching Pokemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return pokemon, fmt.Errorf("Pokemon not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return pokemon, fmt.Errorf("error reading response: %v", err)
	}

	// Add to cache
	cfg.Cache.Add(url, body)

	if err := json.Unmarshal(body, &pokemon); err != nil {
		return pokemon, fmt.Errorf("error parsing JSON: %v", err)
	}

	return pokemon, nil
}

func commandCatch(cfg *config, args ...string) error {
	if len(args) == 0 {
		fmt.Println("Usage: catch <pokemon>")
		return nil
	}

	pokemonName := args[0]
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)

	pokemon, err := fetchPokemon(cfg, pokemonName)
	if err != nil {
		return err
	}

	// Calculate catch chance based on base experience
	// Higher base experience = harder to catch
	// Use a threshold-based system: if random number is less than threshold / (threshold + base_exp), catch succeeds
	threshold := 100.0
	catchChance := threshold / (threshold + float64(pokemon.BaseExperience))

	// Generate random number between 0 and 1
	rand.Seed(time.Now().UnixNano())
	randomValue := rand.Float64()

	if randomValue < catchChance {
		// Caught!
		fmt.Printf("%s was caught!\n", pokemonName)
		cfg.Pokedex[strings.ToLower(pokemonName)] = pokemon
		fmt.Println("You may now inspect it with the inspect command.")
	} else {
		// Escaped!
		fmt.Printf("%s escaped!\n", pokemonName)
	}

	return nil
}

func commandInspect(cfg *config, args ...string) error {
	if len(args) == 0 {
		fmt.Println("Usage: inspect <pokemon>")
		return nil
	}

	pokemonName := strings.ToLower(args[0])
	pokemon, ok := cfg.Pokedex[pokemonName]
	if !ok {
		fmt.Println("you have not caught that pokemon")
		return nil
	}

	fmt.Printf("Name: %s\n", pokemon.Name)
	fmt.Printf("Height: %d\n", pokemon.Height)
	fmt.Printf("Weight: %d\n", pokemon.Weight)
	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf("  -%s: %d\n", stat.Stat.Name, stat.BaseStat)
	}
	fmt.Println("Types:")
	for _, t := range pokemon.Types {
		fmt.Printf("  - %s\n", t.Type.Name)
	}

	return nil
}

func commandPokedex(cfg *config, args ...string) error {
	if len(cfg.Pokedex) == 0 {
		fmt.Println("Your Pokedex is empty.")
		return nil
	}

	fmt.Println("Your Pokedex:")
	for name := range cfg.Pokedex {
		fmt.Printf(" - %s\n", name)
	}

	return nil
}

func main() {
	cfg := &config{
		Cache:   pokecache.NewCache(5 * time.Second),
		Pokedex: make(map[string]Pokemon),
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
		"explore": {
			name:        "explore",
			description: "Lists the Pokemon in a location area",
			callback:    commandExplore,
		},
		"catch": {
			name:        "catch",
			description: "Attempt to catch a Pokemon",
			callback:    commandCatch,
		},
		"inspect": {
			name:        "inspect",
			description: "Display information about a caught Pokemon",
			callback:    commandInspect,
		},
		"pokedex": {
			name:        "pokedex",
			description: "Lists all caught Pokemon",
			callback:    commandPokedex,
		},
	}

	rl, err := readline.New("Pokedex > ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			break
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Add to history (readline handles this automatically, but we can also add it explicitly)
		rl.SaveHistory(input)

		cleaned := cleanInput(input)
		if len(cleaned) > 0 {
			command := cleaned[0]
			args := cleaned[1:]
			if cmd, ok := commands[command]; ok {
				err := cmd.callback(cfg, args...)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				fmt.Println("Unknown command")
			}
		}
	}
}
